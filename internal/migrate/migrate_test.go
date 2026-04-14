package migrate

import (
	"context"
	"errors"
	"strings"
	"testing"

	"bkt2gh/internal/config"
	"bkt2gh/internal/github"
	"bkt2gh/internal/model"
	"bkt2gh/internal/policy"
)

var errFakeClone = errors.New("clone failed")

func TestRunnerDryRunPreflightsWithoutCreatingOrMigrating(t *testing.T) {
	out := new(strings.Builder)
	bb := &fakeBitbucket{
		repos: []model.Repository{
			{Name: "Repo One", Slug: "repo-one", Private: true},
			{Name: "Repo Two", Slug: "repo-two", Private: false},
		},
	}
	gh := &fakeGitHub{existing: map[string]bool{"repo-two": true}}
	git := &fakeGit{}
	runner := Runner{
		Config:    config.Config{BitbucketWorkspace: "team", GitHubOwner: "acme"},
		DryRun:    true,
		Out:       out,
		Bitbucket: bb,
		GitHub:    gh,
		Git:       git,
		SelectRepos: func([]model.Repository) ([]model.Repository, error) {
			return bb.repos, nil
		},
		ChooseVisibility: func() (policy.VisibilityPolicy, error) {
			return policy.AllPrivate, nil
		},
	}

	results, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !bb.called {
		t.Fatal("Bitbucket was not called")
	}
	if !gh.accessChecked {
		t.Fatal("GitHub access was not checked")
	}
	if gh.createCalled {
		t.Fatal("GitHub CreateRepository was called in dry-run")
	}
	if git.called {
		t.Fatal("git migrator was called in dry-run")
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].Status != model.StatusSuccess || results[1].Status != model.StatusSkipped {
		t.Fatalf("results = %+v", results)
	}
	for _, want := range []string{"DRY-RUN migration plan", "repo-one", "would migrate", "repo-two", "would skip"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestRunnerMigratesSelectedRepositories(t *testing.T) {
	bb := &fakeBitbucket{repos: []model.Repository{
		{Name: "Repo One", Slug: "repo-one", Private: true, CloneURL: "bb-url"},
	}}
	gh := &fakeGitHub{cloneURL: "gh-url"}
	git := &fakeGit{}
	runner := Runner{
		Config:           config.Config{BitbucketWorkspace: "team"},
		Bitbucket:        bb,
		GitHub:           gh,
		Git:              git,
		Out:              new(strings.Builder),
		SelectRepos:      func([]model.Repository) ([]model.Repository, error) { return bb.repos, nil },
		ChooseVisibility: func() (policy.VisibilityPolicy, error) { return policy.FollowSource, nil },
	}

	results, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(results) != 1 || results[0].Status != model.StatusSuccess {
		t.Fatalf("results = %+v", results)
	}
	if !gh.request.Private {
		t.Fatal("github request Private = false, want true")
	}
	if git.prepared == nil || git.prepared.dst != "gh-url" {
		t.Fatalf("git push dst = %q", git.prepared.dst)
	}
}

func TestRunnerDoesNotCreateGitHubRepoWhenBitbucketCloneFails(t *testing.T) {
	bb := &fakeBitbucket{repos: []model.Repository{
		{Name: "Repo One", Slug: "repo-one", Private: true, CloneURL: "bb-url"},
	}}
	gh := &fakeGitHub{cloneURL: "gh-url"}
	git := &fakeGit{prepareErr: errFakeClone}
	runner := Runner{
		Config:           config.Config{BitbucketWorkspace: "team"},
		Bitbucket:        bb,
		GitHub:           gh,
		Git:              git,
		Out:              new(strings.Builder),
		SelectRepos:      func([]model.Repository) ([]model.Repository, error) { return bb.repos, nil },
		ChooseVisibility: func() (policy.VisibilityPolicy, error) { return policy.FollowSource, nil },
	}

	results, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if gh.createCalled {
		t.Fatal("GitHub CreateRepository was called after Bitbucket clone failure")
	}
	if len(results) != 1 || results[0].Status != model.StatusFailed {
		t.Fatalf("results = %+v, want one failed result", results)
	}
}

type fakeBitbucket struct {
	called bool
	repos  []model.Repository
}

func (f *fakeBitbucket) ListRepositories(ctx context.Context, workspace string) ([]model.Repository, error) {
	f.called = true
	return f.repos, nil
}

type fakeGitHub struct {
	request       github.CreateRepositoryRequest
	cloneURL      string
	existing      map[string]bool
	accessChecked bool
	createCalled  bool
}

func (f *fakeGitHub) CreateRepository(ctx context.Context, req github.CreateRepositoryRequest) (github.CreateRepositoryResult, error) {
	f.createCalled = true
	f.request = req
	return github.CreateRepositoryResult{CloneURL: f.cloneURL}, nil
}

func (f *fakeGitHub) RepositoryExists(ctx context.Context, name string) (bool, error) {
	return f.existing[name], nil
}

func (f *fakeGitHub) CheckCreateAccess(ctx context.Context) error {
	f.accessChecked = true
	return nil
}

type fakeGit struct {
	called     bool
	prepareErr error
	prepared   *fakePreparedMirror
}

func (f *fakeGit) Prepare(ctx context.Context, repo model.Repository) (interface {
	Push(ctx context.Context, githubCloneURL string) error
	Cleanup() error
}, error) {
	f.called = true
	if f.prepareErr != nil {
		return nil, f.prepareErr
	}
	f.prepared = &fakePreparedMirror{}
	return f.prepared, nil
}

type fakePreparedMirror struct {
	dst string
}

func (f *fakePreparedMirror) Push(ctx context.Context, githubCloneURL string) error {
	f.dst = githubCloneURL
	return nil
}

func (f *fakePreparedMirror) Cleanup() error {
	return nil
}
