package gitops

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"bkt2gh/internal/model"
)

type Runner interface {
	Run(ctx context.Context, dir string, env []string, name string, args ...string) error
}

type ExecRunner struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (r ExecRunner) Run(ctx context.Context, dir string, env []string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v failed: %w", name, args, err)
	}
	return nil
}

type MirrorMigrator struct {
	Runner               Runner
	TempDir              string
	Out                  io.Writer
	BitbucketUsername    string
	BitbucketAppPassword string
	GitHubUsername       string
	GitHubToken          string
	GitLFSAvailable      func() bool
}

type PreparedMirror struct {
	Runner    Runner
	MirrorDir string
	Out       io.Writer
	PushEnv   []string
	UseLFS    bool
	cleanup   func() error
}

func (m MirrorMigrator) Prepare(ctx context.Context, repo model.Repository) (interface {
	Push(ctx context.Context, githubCloneURL string) error
	Cleanup() error
}, error) {
	if repo.CloneURL == "" {
		return nil, fmt.Errorf("Bitbucket clone URL missing for %s", repo.Slug)
	}
	runner := m.Runner
	if runner == nil {
		runner = ExecRunner{Stdout: m.Out, Stderr: m.Out}
	}
	baseDir := m.TempDir
	cleanup := func() error { return nil }
	if baseDir == "" {
		var err error
		baseDir, err = os.MkdirTemp("", "bkt2gh-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp directory: %w", err)
		}
		cleanup = func() error { return os.RemoveAll(baseDir) }
	}
	bitbucketEnv, err := m.gitAuthEnv(baseDir, "bitbucket", "BKT2GH_BITBUCKET_USERNAME", m.BitbucketUsername, "BKT2GH_BITBUCKET_APP_PASSWORD", m.BitbucketAppPassword)
	if err != nil {
		cleanup()
		return nil, err
	}
	githubEnv, err := m.gitAuthEnv(baseDir, "github", "BKT2GH_GITHUB_USERNAME", githubUsername(m.GitHubUsername), "BKT2GH_GITHUB_TOKEN", m.GitHubToken)
	if err != nil {
		cleanup()
		return nil, err
	}
	useLFS := m.gitLFSAvailable()

	mirrorDir := filepath.Join(baseDir, repo.Slug+".git")
	if err := os.RemoveAll(mirrorDir); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to remove existing mirror clone for %s: %w", repo.Slug, err)
	}
	if err := runner.Run(ctx, baseDir, bitbucketEnv, "git", "clone", "--mirror", repo.CloneURL, mirrorDir); err != nil {
		cleanup()
		return nil, err
	}
	if useLFS {
		if err := runner.Run(ctx, mirrorDir, bitbucketEnv, "git", "lfs", "fetch", "--all"); err != nil && m.Out != nil {
			fmt.Fprintf(m.Out, "LFS fetch skipped for %s: %v\n", repo.Slug, err)
		}
	} else if m.Out != nil {
		fmt.Fprintf(m.Out, "LFS fetch skipped for %s: git-lfs is not available\n", repo.Slug)
	}
	return &PreparedMirror{
		Runner:    runner,
		MirrorDir: mirrorDir,
		Out:       m.Out,
		PushEnv:   githubEnv,
		UseLFS:    useLFS,
		cleanup:   cleanup,
	}, nil
}

func (m MirrorMigrator) gitAuthEnv(baseDir, service, usernameEnv, username, passwordEnv, password string) ([]string, error) {
	if username == "" || password == "" {
		return nil, nil
	}
	askPassPath := filepath.Join(baseDir, service+"-askpass.sh")
	script := fmt.Sprintf(`#!/bin/sh
case "$1" in
*Username*) printf '%%s\n' "$%s" ;;
*Password*) printf '%%s\n' "$%s" ;;
*) exit 1 ;;
esac
`, usernameEnv, passwordEnv)
	if err := os.WriteFile(askPassPath, []byte(script), 0o700); err != nil {
		return nil, fmt.Errorf("failed to write %s askpass helper: %w", service, err)
	}
	return []string{
		"GIT_ASKPASS=" + askPassPath,
		"GIT_TERMINAL_PROMPT=0",
		usernameEnv + "=" + username,
		passwordEnv + "=" + password,
	}, nil
}

func (m MirrorMigrator) gitLFSAvailable() bool {
	if m.GitLFSAvailable != nil {
		return m.GitLFSAvailable()
	}
	_, err := exec.LookPath("git-lfs")
	return err == nil
}

func githubUsername(username string) string {
	if username != "" {
		return username
	}
	return "x-access-token"
}

func (m MirrorMigrator) Migrate(ctx context.Context, repo model.Repository, githubCloneURL string) error {
	prepared, err := m.Prepare(ctx, repo)
	if err != nil {
		return err
	}
	defer prepared.Cleanup()
	return prepared.Push(ctx, githubCloneURL)
}

func (m *PreparedMirror) Push(ctx context.Context, githubCloneURL string) error {
	if githubCloneURL == "" {
		return fmt.Errorf("GitHub clone URL missing")
	}
	if err := m.Runner.Run(ctx, m.MirrorDir, m.PushEnv, "git", "remote", "set-url", "origin", githubCloneURL); err != nil {
		return err
	}
	if m.UseLFS {
		if err := m.Runner.Run(ctx, m.MirrorDir, m.PushEnv, "git", "lfs", "push", "--all", "origin"); err != nil && m.Out != nil {
			fmt.Fprintf(m.Out, "LFS push skipped: %v\n", err)
		}
	} else if m.Out != nil {
		fmt.Fprintln(m.Out, "LFS push skipped: git-lfs is not available")
	}
	if err := m.Runner.Run(ctx, m.MirrorDir, m.PushEnv, "git", "push", "--mirror", "origin"); err != nil {
		return err
	}
	return nil
}

func (m *PreparedMirror) Cleanup() error {
	if m.cleanup == nil {
		return nil
	}
	return m.cleanup()
}

type DryRunMigrator struct {
	Out io.Writer
}

func (m DryRunMigrator) Migrate(ctx context.Context, repo model.Repository, githubCloneURL string) error {
	if m.Out != nil {
		fmt.Fprintf(m.Out, "DRY-RUN git: clone --mirror %s; push --mirror %s\n", repo.CloneURL, githubCloneURL)
	}
	return nil
}
