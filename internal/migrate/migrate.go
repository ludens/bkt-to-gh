package migrate

import (
	"context"
	"fmt"
	"io"

	"github.com/ludens/bkt-to-gh/internal/config"
	"github.com/ludens/bkt-to-gh/internal/github"
	"github.com/ludens/bkt-to-gh/internal/model"
	"github.com/ludens/bkt-to-gh/internal/policy"
)

type BitbucketClient interface {
	ListRepositories(ctx context.Context, workspace string) ([]model.Repository, error)
}

type GitHubClient interface {
	CreateRepository(ctx context.Context, req github.CreateRepositoryRequest) (github.CreateRepositoryResult, error)
	RepositoryExists(ctx context.Context, name string) (bool, error)
	CheckCreateAccess(ctx context.Context) error
}

type GitMigrator interface {
	Prepare(ctx context.Context, repo model.Repository) (interface {
		Push(ctx context.Context, githubCloneURL string) error
		Cleanup() error
	}, error)
}

type Runner struct {
	Config           config.Config
	DryRun           bool
	Out              io.Writer
	Bitbucket        BitbucketClient
	GitHub           GitHubClient
	Git              GitMigrator
	SelectRepos      func([]model.Repository) ([]model.Repository, error)
	ChooseVisibility func() (policy.VisibilityPolicy, error)
}

func (r Runner) Run(ctx context.Context) ([]model.RepoResult, error) {
	out := r.out()
	if r.Bitbucket == nil || r.GitHub == nil || (!r.DryRun && r.Git == nil) {
		return nil, fmt.Errorf("migrate runner is missing required clients")
	}

	fmt.Fprintf(out, "Fetching Bitbucket repositories from workspace %q...\n", r.Config.BitbucketWorkspace)
	repos, err := r.Bitbucket.ListRepositories(ctx, r.Config.BitbucketWorkspace)
	if err != nil {
		return nil, err
	}
	if len(repos) == 0 {
		fmt.Fprintln(out, "No repositories found.")
		return nil, nil
	}

	selectRepos := r.SelectRepos
	if selectRepos == nil {
		return nil, fmt.Errorf("repository selector is not configured")
	}
	selected, err := selectRepos(repos)
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		fmt.Fprintln(out, "No repositories selected.")
		return nil, nil
	}

	chooseVisibility := r.ChooseVisibility
	if chooseVisibility == nil {
		return nil, fmt.Errorf("visibility policy prompt is not configured")
	}
	visibilityPolicy, err := chooseVisibility()
	if err != nil {
		return nil, err
	}
	if r.DryRun {
		return r.runDryRun(ctx, selected, visibilityPolicy)
	}

	results := []model.RepoResult{}
	for _, repo := range selected {
		results = append(results, r.migrateOne(ctx, repo, visibilityPolicy))
	}
	PrintSummary(out, results)
	return results, nil
}

func (r Runner) migrateOne(ctx context.Context, repo model.Repository, visibilityPolicy policy.VisibilityPolicy) model.RepoResult {
	out := r.out()
	fmt.Fprintf(out, "Migrating %s...\n", repo.Slug)
	private, err := policy.ResolveVisibility(visibilityPolicy, repo.Private)
	if err != nil {
		return model.RepoResult{Repo: repo, Status: model.StatusFailed, Reason: err.Error()}
	}
	prepared, err := r.Git.Prepare(ctx, repo)
	if err != nil {
		return model.RepoResult{Repo: repo, Status: model.StatusFailed, Reason: err.Error()}
	}
	cleanup := func() error {
		if err := prepared.Cleanup(); err != nil {
			return fmt.Errorf("cleanup failed for %s: %w", repo.Slug, err)
		}
		return nil
	}
	created, err := r.GitHub.CreateRepository(ctx, github.CreateRepositoryRequest{
		Name:        repo.Slug,
		Description: repo.Description,
		Private:     private,
	})
	if err != nil {
		if cleanupErr := cleanup(); cleanupErr != nil {
			return model.RepoResult{Repo: repo, Status: model.StatusFailed, Reason: err.Error() + "; " + cleanupErr.Error()}
		}
		return model.RepoResult{Repo: repo, Status: model.StatusFailed, Reason: err.Error()}
	}
	if created.Skipped {
		fmt.Fprintf(out, "Skip %s: %s\n", repo.Slug, created.Reason)
		if cleanupErr := cleanup(); cleanupErr != nil {
			return model.RepoResult{Repo: repo, Status: model.StatusFailed, Reason: cleanupErr.Error()}
		}
		return model.RepoResult{Repo: repo, Status: model.StatusSkipped, Reason: created.Reason}
	}
	if err := prepared.Push(ctx, created.CloneURL); err != nil {
		if cleanupErr := cleanup(); cleanupErr != nil {
			return model.RepoResult{Repo: repo, Status: model.StatusFailed, Reason: err.Error() + "; " + cleanupErr.Error()}
		}
		return model.RepoResult{Repo: repo, Status: model.StatusFailed, Reason: err.Error()}
	}
	if cleanupErr := cleanup(); cleanupErr != nil {
		return model.RepoResult{Repo: repo, Status: model.StatusFailed, Reason: cleanupErr.Error()}
	}
	return model.RepoResult{Repo: repo, Status: model.StatusSuccess}
}

func (r Runner) runDryRun(ctx context.Context, selected []model.Repository, visibilityPolicy policy.VisibilityPolicy) ([]model.RepoResult, error) {
	out := r.out()
	fmt.Fprintln(out, "Checking GitHub token and owner access...")
	if err := r.GitHub.CheckCreateAccess(ctx); err != nil {
		return nil, err
	}

	rows := []planRow{}
	results := []model.RepoResult{}
	for _, repo := range selected {
		private, err := policy.ResolveVisibility(visibilityPolicy, repo.Private)
		if err != nil {
			results = append(results, model.RepoResult{Repo: repo, Status: model.StatusFailed, Reason: err.Error()})
			rows = append(rows, planRow{Repo: repo, TargetPrivate: private, GitHubStatus: "error", Action: "would fail", Reason: err.Error()})
			continue
		}
		exists, err := r.GitHub.RepositoryExists(ctx, repo.Slug)
		if err != nil {
			results = append(results, model.RepoResult{Repo: repo, Status: model.StatusFailed, Reason: err.Error()})
			rows = append(rows, planRow{Repo: repo, TargetPrivate: private, GitHubStatus: "error", Action: "would fail", Reason: err.Error()})
			continue
		}
		if exists {
			reason := "GitHub repository already exists; overwrite is disabled"
			results = append(results, model.RepoResult{Repo: repo, Status: model.StatusSkipped, Reason: reason})
			rows = append(rows, planRow{Repo: repo, TargetPrivate: private, GitHubStatus: "exists", Action: "would skip", Reason: reason})
			continue
		}
		results = append(results, model.RepoResult{Repo: repo, Status: model.StatusSuccess})
		rows = append(rows, planRow{Repo: repo, TargetPrivate: private, GitHubStatus: "available", Action: "would migrate"})
	}
	PrintDryRunPlan(out, rows)
	PrintSummary(out, results)
	return results, nil
}

type planRow struct {
	Repo          model.Repository
	TargetPrivate bool
	GitHubStatus  string
	Action        string
	Reason        string
}

func PrintDryRunPlan(out io.Writer, rows []planRow) {
	fmt.Fprintln(out, "Migration preview")
	fmt.Fprintf(out, "%-24s %-18s %-18s %-14s %-14s %s\n", "repo", "source visibility", "target visibility", "github", "action", "reason")
	fmt.Fprintf(out, "%-24s %-18s %-18s %-14s %-14s %s\n", "----", "-----------------", "-----------------", "------", "------", "------")
	for _, row := range rows {
		fmt.Fprintf(out, "%-24s %-18s %-18s %-14s %-14s %s\n",
			row.Repo.Slug,
			visibilityLabel(row.Repo.Private),
			visibilityLabel(row.TargetPrivate),
			row.GitHubStatus,
			row.Action,
			row.Reason,
		)
	}
}

func PrintSummary(out io.Writer, results []model.RepoResult) {
	counts := map[model.RepoStatus]int{}
	for _, result := range results {
		counts[result.Status]++
	}
	fmt.Fprintln(out, "Summary:")
	fmt.Fprintf(out, "  success: %d\n", counts[model.StatusSuccess])
	fmt.Fprintf(out, "  failed: %d\n", counts[model.StatusFailed])
	fmt.Fprintf(out, "  skipped: %d\n", counts[model.StatusSkipped])
	for _, result := range results {
		if result.Status != model.StatusSuccess {
			fmt.Fprintf(out, "  %s: %s (%s)\n", result.Status, result.Repo.Slug, result.Reason)
		}
	}
}

func (r Runner) out() io.Writer {
	if r.Out != nil {
		return r.Out
	}
	return io.Discard
}

func visibilityLabel(private bool) string {
	if private {
		return "private"
	}
	return "public"
}
