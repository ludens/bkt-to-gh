package gitops

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ludens/bkt-to-gh/internal/model"
)

func TestMirrorMigratorRunsMirrorCloneLFSAndPush(t *testing.T) {
	runner := &recordingRunner{}
	migrator := MirrorMigrator{
		Runner:          runner,
		TempDir:         t.TempDir(),
		GitLFSAvailable: func() bool { return true },
	}

	err := migrator.Migrate(context.Background(), model.Repository{
		Slug:     "repo-one",
		CloneURL: "https://bitbucket.org/team/repo-one.git",
	}, "https://github.com/acme/repo-one.git")
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	joined := runner.joinedArgs()
	for _, want := range []string{
		"git clone --mirror https://bitbucket.org/team/repo-one.git",
		"git lfs fetch --all",
		"git remote set-url origin https://github.com/acme/repo-one.git",
		"git lfs push --all origin",
		"git push --mirror origin",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("calls missing %q in:\n%s", want, joined)
		}
	}
}

func TestDryRunMigratorDoesNotRunCommands(t *testing.T) {
	runner := &recordingRunner{}
	migrator := DryRunMigrator{Out: new(strings.Builder)}

	if err := migrator.Migrate(context.Background(), model.Repository{Slug: "repo"}, "https://github.com/acme/repo.git"); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %v, want none", runner.calls)
	}
}

func TestMirrorMigratorRemovesExistingMirrorBeforeClone(t *testing.T) {
	tempDir := t.TempDir()
	existingMirror := filepath.Join(tempDir, "repo-one.git")
	if err := os.MkdirAll(existingMirror, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(existingMirror, "stale"), []byte("old clone"), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := &cloneTargetMustNotExistRunner{t: t}
	migrator := MirrorMigrator{
		Runner:          runner,
		TempDir:         tempDir,
		GitLFSAvailable: func() bool { return true },
	}

	err := migrator.Migrate(context.Background(), model.Repository{
		Slug:     "repo-one",
		CloneURL: "https://bitbucket.org/team/repo-one.git",
	}, "https://github.com/acme/repo-one.git")
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if !runner.sawClone {
		t.Fatal("clone command was not run")
	}
}

func TestMirrorMigratorUsesAskPassForBitbucketHTTPSClone(t *testing.T) {
	runner := &recordingRunner{}
	migrator := MirrorMigrator{
		Runner:               runner,
		TempDir:              t.TempDir(),
		BitbucketUsername:    "alice",
		BitbucketAppPassword: "secret-app-password",
		GitLFSAvailable:      func() bool { return true },
	}

	err := migrator.Migrate(context.Background(), model.Repository{
		Slug:     "repo-one",
		CloneURL: "https://bitbucket.org/team/repo-one.git",
	}, "https://github.com/acme/repo-one.git")
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if len(runner.calls) == 0 {
		t.Fatal("runner calls = 0, want git clone call")
	}
	clone := runner.calls[0]
	if strings.Contains(clone.args, "secret-app-password") {
		t.Fatalf("clone args leaked app password: %s", clone.args)
	}
	for _, want := range []string{
		"GIT_ASKPASS=",
		"GIT_TERMINAL_PROMPT=0",
		"BKT2GH_BITBUCKET_USERNAME=alice",
		"BKT2GH_BITBUCKET_APP_PASSWORD=secret-app-password",
	} {
		if !containsEnv(clone.env, want) {
			t.Fatalf("clone env missing %q in %v", want, clone.env)
		}
	}
}

func TestMirrorMigratorSkipsLFSWhenGitLFSIsUnavailable(t *testing.T) {
	runner := &recordingRunner{}
	out := new(strings.Builder)
	migrator := MirrorMigrator{
		Runner:          runner,
		TempDir:         t.TempDir(),
		Out:             out,
		GitLFSAvailable: func() bool { return false },
	}

	err := migrator.Migrate(context.Background(), model.Repository{
		Slug:     "repo-one",
		CloneURL: "https://bitbucket.org/team/repo-one.git",
	}, "https://github.com/acme/repo-one.git")
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	joined := runner.joinedArgs()
	if strings.Contains(joined, "git lfs") {
		t.Fatalf("git lfs was called despite unavailable git-lfs:\n%s", joined)
	}
	if !strings.Contains(out.String(), "git-lfs is not available") {
		t.Fatalf("output missing git-lfs unavailable warning: %q", out.String())
	}
}

func TestPreparedMirrorUsesAskPassForGitHubPush(t *testing.T) {
	runner := &recordingRunner{}
	migrator := MirrorMigrator{
		Runner:          runner,
		TempDir:         t.TempDir(),
		GitHubUsername:  "acme",
		GitHubToken:     "ghp-secret",
		GitLFSAvailable: func() bool { return true },
	}

	err := migrator.Migrate(context.Background(), model.Repository{
		Slug:     "repo-one",
		CloneURL: "https://bitbucket.org/team/repo-one.git",
	}, "https://github.com/acme/repo-one.git")
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	var push recordedCall
	for _, call := range runner.calls {
		if strings.Contains(call.args, "git push --mirror origin") {
			push = call
			break
		}
	}
	if push.args == "" {
		t.Fatalf("push call missing in:\n%s", runner.joinedArgs())
	}
	if strings.Contains(push.args, "ghp-secret") {
		t.Fatalf("push args leaked GitHub token: %s", push.args)
	}
	for _, want := range []string{
		"GIT_ASKPASS=",
		"GIT_TERMINAL_PROMPT=0",
		"BKT2GH_GITHUB_USERNAME=acme",
		"BKT2GH_GITHUB_TOKEN=ghp-secret",
	} {
		if !containsEnv(push.env, want) {
			t.Fatalf("push env missing %q in %v", want, push.env)
		}
	}
}

type recordingRunner struct {
	calls []recordedCall
}

type recordedCall struct {
	env  []string
	args string
}

func (r *recordingRunner) Run(ctx context.Context, dir string, env []string, name string, args ...string) error {
	r.calls = append(r.calls, recordedCall{env: env, args: name + " " + strings.Join(args, " ")})
	return nil
}

func (r *recordingRunner) joinedArgs() string {
	lines := make([]string, 0, len(r.calls))
	for _, call := range r.calls {
		lines = append(lines, call.args)
	}
	return strings.Join(lines, "\n")
}

type cloneTargetMustNotExistRunner struct {
	t        *testing.T
	sawClone bool
}

func (r *cloneTargetMustNotExistRunner) Run(ctx context.Context, dir string, env []string, name string, args ...string) error {
	if len(args) >= 4 && name == "git" && args[0] == "clone" && args[1] == "--mirror" {
		r.sawClone = true
		target := args[3]
		if _, err := os.Stat(target); err == nil {
			r.t.Fatalf("clone target %s exists before git clone", target)
		} else if !os.IsNotExist(err) {
			r.t.Fatalf("stat clone target %s: %v", target, err)
		}
	}
	return nil
}

func containsEnv(env []string, prefix string) bool {
	for _, value := range env {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
