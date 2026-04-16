package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ludens/bkt-to-gh/internal/config"
)

func TestRunRootHelpReturnsNil(t *testing.T) {
	for _, args := range [][]string{
		{"--help"},
		{"-h"},
		{"help"},
	} {
		t.Run(args[0], func(t *testing.T) {
			if err := run(args); err != nil {
				t.Fatalf("run(%v) error = %v, want nil", args, err)
			}
		})
	}
}

func TestRunCommandHelpReturnsNil(t *testing.T) {
	for _, args := range [][]string{
		{"migrate", "--help"},
		{"migrate-preview", "--help"},
		{"configure", "--help"},
	} {
		t.Run(args[0], func(t *testing.T) {
			if err := run(args); err != nil {
				t.Fatalf("run(%v) error = %v, want nil", args, err)
			}
		})
	}
}

func TestMigrateConfigureFlagIsRemoved(t *testing.T) {
	if err := run([]string{"migrate", "--configure"}); err == nil {
		t.Fatal("run([migrate --configure]) error = nil, want error")
	}
}

func TestMigrateDryRunFlagIsRemoved(t *testing.T) {
	err := run([]string{"migrate", "--dry-run"})
	if err == nil {
		t.Fatal("run([migrate --dry-run]) error = nil, want usage error")
	}
	if !errors.Is(err, errUsage) {
		t.Fatalf("run([migrate --dry-run]) error = %v, want usage error", err)
	}
}

func TestRunCLIInvalidCommandReturnsUsageExitCode(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder

	code := runCLI(context.Background(), strings.NewReader(""), &stdout, &stderr, []string{"wat"})

	if code != 2 {
		t.Fatalf("runCLI() code = %d, want 2", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), `Error: unknown command "wat"`) {
		t.Fatalf("stderr missing unknown command error: %q", stderr.String())
	}
}

func TestRunCLIMigrateFlagErrorUsesStderrOnly(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder

	code := runCLI(context.Background(), strings.NewReader(""), &stdout, &stderr, []string{"migrate", "--definitely-not-a-flag"})

	if code != 2 {
		t.Fatalf("runCLI() code = %d, want 2", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("stderr missing flag parse error: %q", stderr.String())
	}
}

func TestRunCLIRejectsUnexpectedPositionalArgs(t *testing.T) {
	tests := [][]string{
		{"configure", "extra"},
		{"migrate", "extra"},
		{"migrate-preview", "extra"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var stdout strings.Builder
			var stderr strings.Builder

			code := runCLI(context.Background(), strings.NewReader(""), &stdout, &stderr, args)

			if code != 2 {
				t.Fatalf("runCLI(%v) code = %d, want 2", args, code)
			}
			if stdout.String() != "" {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if !strings.Contains(stderr.String(), "unexpected argument") {
				t.Fatalf("stderr missing unexpected argument: %q", stderr.String())
			}
		})
	}
}

func TestRunCLICommandHelpUsesStdoutOnly(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "configure", args: []string{"configure", "--help"}, want: "bkt2gh configure"},
		{name: "migrate", args: []string{"migrate", "--workspace", "team", "--help"}, want: "bkt2gh migrate [--workspace name]"},
		{name: "migrate-preview", args: []string{"migrate-preview", "--workspace", "team", "--help"}, want: "bkt2gh migrate-preview [--workspace name]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout strings.Builder
			var stderr strings.Builder

			code := runCLI(context.Background(), strings.NewReader(""), &stdout, &stderr, tt.args)

			if code != 0 {
				t.Fatalf("runCLI(%v) code = %d, want 0", tt.args, code)
			}
			if !strings.Contains(stdout.String(), tt.want) {
				t.Fatalf("stdout missing %q:\n%s", tt.want, stdout.String())
			}
			if stderr.String() != "" {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestRunConfigureWritesEncryptedConfigToDefaultPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	keyring := newMainTestKeyring()
	withTestConfigStore(t, path, keyring)

	input := strings.NewReader(strings.Join([]string{
		"person@example.com",
		"bb-app-password",
		"team",
		"gh-token",
		"acme",
		"",
	}, "\n"))

	err := runWithIO(context.Background(), input, new(strings.Builder), new(strings.Builder), []string{"configure"})
	if err != nil {
		t.Fatalf("runWithIO(configure) error = %v", err)
	}

	cfg, err := config.Load(path, keyring)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if cfg.BitbucketWorkspace != "team" {
		t.Fatalf("BitbucketWorkspace = %q, want team", cfg.BitbucketWorkspace)
	}
}

func TestRunMigrateUsesProvidedContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runWithIO(ctx, strings.NewReader(""), new(strings.Builder), new(strings.Builder), []string{"migrate", "--workspace", "team"})

	if err == nil {
		t.Fatal("runWithIO() error = nil, want context cancellation or configuration error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runWithIO() error = %v, want context cancellation before migration work", err)
	}
}

func TestRunMigratePreviewUsesProvidedContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runWithIO(ctx, strings.NewReader(""), new(strings.Builder), new(strings.Builder), []string{"migrate-preview", "--workspace", "team"})

	if err == nil {
		t.Fatal("runWithIO() error = nil, want context cancellation or configuration error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runWithIO() error = %v, want context cancellation before migration work", err)
	}
}

type mainTestKeyring struct {
	values map[string]string
}

func newMainTestKeyring() *mainTestKeyring {
	return &mainTestKeyring{values: map[string]string{}}
}

func (m *mainTestKeyring) Get(service, user string) (string, error) {
	value, ok := m.values[service+"/"+user]
	if !ok {
		return "", config.ErrKeyNotFound
	}
	return value, nil
}

func (m *mainTestKeyring) Set(service, user, password string) error {
	m.values[service+"/"+user] = password
	return nil
}

func withTestConfigStore(t *testing.T, path string, keyring config.Keyring) {
	t.Helper()
	oldPath := defaultConfigPath
	oldKeyring := defaultKeyring
	defaultConfigPath = func() (string, error) { return path, nil }
	defaultKeyring = func() config.Keyring { return keyring }
	t.Cleanup(func() {
		defaultConfigPath = oldPath
		defaultKeyring = oldKeyring
	})
}
