package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/ludens/bkt-to-gh/internal/bitbucket"
	"github.com/ludens/bkt-to-gh/internal/config"
	"github.com/ludens/bkt-to-gh/internal/github"
	"github.com/ludens/bkt-to-gh/internal/gitops"
	"github.com/ludens/bkt-to-gh/internal/migrate"
	"github.com/ludens/bkt-to-gh/internal/model"
	"github.com/ludens/bkt-to-gh/internal/policy"
	"github.com/ludens/bkt-to-gh/internal/prompt"
)

var errUsage = errors.New("usage error")

var defaultConfigPath = config.DefaultPath
var defaultKeyring = config.DefaultKeyring

type usageError struct {
	err error
}

func (e usageError) Error() string {
	return e.err.Error()
}

func (e usageError) Unwrap() error {
	return errUsage
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(runCLI(ctx, os.Stdin, os.Stdout, os.Stderr, os.Args[1:]))
}

func runCLI(ctx context.Context, in io.Reader, out, errOut io.Writer, args []string) int {
	if err := runWithIO(ctx, in, out, errOut, args); err != nil {
		fmt.Fprintln(errOut, "Error:", err)
		if errors.Is(err, errUsage) {
			return 2
		}
		return 1
	}
	return 0
}

func run(args []string) error {
	return runWithIO(context.Background(), os.Stdin, os.Stdout, os.Stderr, args)
}

func runWithIO(ctx context.Context, in io.Reader, out, errOut io.Writer, args []string) error {
	if len(args) == 0 || isHelp(args[0]) {
		printUsage(out)
		return nil
	}
	switch args[0] {
	case "help":
		printUsage(out)
		return nil
	case "configure":
		return runConfigure(in, out, errOut, args[1:])
	case "migrate-preview":
		return runMigratePreview(ctx, in, out, errOut, args[1:])
	case "migrate":
		return runMigrate(ctx, in, out, errOut, args[1:])
	default:
		return usageError{err: fmt.Errorf("unknown command %q", args[0])}
	}
}

func runConfigure(in io.Reader, out, errOut io.Writer, args []string) error {
	if len(args) > 0 && isHelp(args[0]) {
		printConfigureUsage(out)
		return nil
	}
	fs := flag.NewFlagSet("configure", flag.ContinueOnError)
	fs.SetOutput(errOut)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return usageError{err: err}
	}
	configPath, err := defaultConfigPath()
	if err != nil {
		return err
	}
	_, _, err = config.ConfigureInteractiveIfAllowed(in, out, configPath, defaultKeyring())
	return err
}

func runMigrate(ctx context.Context, in io.Reader, out, errOut io.Writer, args []string) error {
	if len(args) > 0 && isHelp(args[0]) {
		printMigrateUsage(out)
		return nil
	}
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.SetOutput(errOut)
	workspace := fs.String("workspace", "", "Bitbucket workspace")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return usageError{err: err}
	}
	return runMigration(ctx, in, out, *workspace, false)
}

func runMigratePreview(ctx context.Context, in io.Reader, out, errOut io.Writer, args []string) error {
	if len(args) > 0 && isHelp(args[0]) {
		printMigratePreviewUsage(out)
		return nil
	}
	fs := flag.NewFlagSet("migrate-preview", flag.ContinueOnError)
	fs.SetOutput(errOut)
	workspace := fs.String("workspace", "", "Bitbucket workspace")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return usageError{err: err}
	}
	return runMigration(ctx, in, out, *workspace, true)
}

func runMigration(ctx context.Context, in io.Reader, out io.Writer, workspace string, dryRun bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	configPath, err := defaultConfigPath()
	if err != nil {
		return err
	}
	keyring := defaultKeyring()
	if !config.HasConfig(configPath) {
		fmt.Fprintf(out, "%s not found. Starting setup. You can also run `bkt2gh configure` anytime.\n", configPath)
		if _, err := config.ConfigureInteractive(in, out, configPath, keyring); err != nil {
			return err
		}
	}

	cfg, err := config.Load(configPath, keyring)
	if err != nil {
		return err
	}
	if workspace != "" {
		cfg.BitbucketWorkspace = workspace
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	bb := bitbucket.NewClient(cfg.BitbucketUsername, cfg.BitbucketAppPassword)
	gh := github.NewClient(cfg.GitHubToken, cfg.GitHubOwner)
	git := gitops.MirrorMigrator{
		Out:                  out,
		BitbucketUsername:    cfg.BitbucketUsername,
		BitbucketAppPassword: cfg.BitbucketAppPassword,
		GitHubUsername:       cfg.GitHubOwner,
		GitHubToken:          cfg.GitHubToken,
	}
	runner := migrate.Runner{
		Config:    cfg,
		DryRun:    dryRun,
		Out:       out,
		Bitbucket: bb,
		GitHub:    gh,
		Git:       git,
		SelectRepos: func(repos []model.Repository) ([]model.Repository, error) {
			return selectRepositories(in, out, repos)
		},
		ChooseVisibility: func() (policy.VisibilityPolicy, error) {
			return chooseVisibilityPolicy(in, out)
		},
	}
	_, err = runner.Run(ctx)
	return err
}

func selectRepositories(in io.Reader, out io.Writer, repos []model.Repository) ([]model.Repository, error) {
	inFile, inOK := in.(*os.File)
	outFile, outOK := out.(*os.File)
	if inOK && outOK {
		return prompt.SelectRepositoriesAuto(inFile, outFile, repos)
	}
	return prompt.SelectRepositories(in, out, repos)
}

func chooseVisibilityPolicy(in io.Reader, out io.Writer) (policy.VisibilityPolicy, error) {
	inFile, inOK := in.(*os.File)
	outFile, outOK := out.(*os.File)
	if inOK && outOK {
		return prompt.ChooseVisibilityPolicyAuto(inFile, outFile)
	}
	return prompt.ChooseVisibilityPolicy(in, out)
}

func printUsage(out io.Writer) {
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  bkt2gh configure")
	fmt.Fprintln(out, "  bkt2gh migrate-preview [--workspace name]")
	fmt.Fprintln(out, "  bkt2gh migrate [--workspace name]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Commands:")
	fmt.Fprintln(out, "  configure        create or update encrypted config.yaml interactively")
	fmt.Fprintln(out, "  migrate-preview  preview migration plan without creating or pushing")
	fmt.Fprintln(out, "  migrate          migrate selected Bitbucket repositories to GitHub")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Flags:")
	fmt.Fprintln(out, "  -h, --help show help")
}

func printConfigureUsage(out io.Writer) {
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  bkt2gh configure")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Creates or updates encrypted config.yaml interactively.")
}

func printMigrateUsage(out io.Writer) {
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  bkt2gh migrate [--workspace name]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Flags:")
	fmt.Fprintln(out, "  --workspace name  Bitbucket workspace")
	fmt.Fprintln(out, "  -h, --help        show help")
}

func printMigratePreviewUsage(out io.Writer) {
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  bkt2gh migrate-preview [--workspace name]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Preflights Bitbucket/GitHub and prints the migration plan without creating or pushing.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Flags:")
	fmt.Fprintln(out, "  --workspace name  Bitbucket workspace")
	fmt.Fprintln(out, "  -h, --help        show help")
}

func isHelp(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "help"
}
