# bkt2gh CLI Test Quality Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 리뷰에서 나온 CLI 동작 문제와 의미 낮은 테스트를 정리해, 실패할 수 있는 테스트만 남기고 실제 사용자 동작을 더 엄격하게 검증한다.

**Architecture:** 기존 stdlib `flag` 기반 command 구조를 유지한다. CLI boundary는 usage/help/exit code/stdout-stderr를 담당하고, migration orchestration은 `internal/migrate`, git 작업은 `internal/gitops`, API 호출은 각 client package에 남긴다.

**Tech Stack:** Go 1.22, standard library `flag`, `httptest`, 기존 fake client/runner 테스트 패턴.

---

### Task 1: CLI extra args and help output

**Files:**
- Modify: `cmd/bkt2gh/main.go`
- Modify: `cmd/bkt2gh/main_test.go`

- [x] **Step 1: Add failing tests for unexpected positional args**

Add to `cmd/bkt2gh/main_test.go`:

```go
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
```

- [x] **Step 2: Add failing tests for custom help anywhere in subcommand args**

Add to `cmd/bkt2gh/main_test.go`:

```go
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
```

- [x] **Step 3: Run targeted tests and verify failure**

Run:

```bash
env GOPATH=/tmp/go GOMODCACHE=/tmp/go/pkg/mod GOCACHE=/tmp/go-build go test ./cmd/bkt2gh -run 'TestRunCLIRejectsUnexpectedPositionalArgs|TestRunCLICommandHelpUsesStdoutOnly'
```

Expected: FAIL because current implementation ignores `fs.Args()` and lets `flag` print help to stderr when `--help` is not the first subcommand arg.

- [x] **Step 4: Implement CLI parsing helpers**

Add to `cmd/bkt2gh/main.go` near `isHelp`:

```go
func hasHelpArg(args []string) bool {
	for _, arg := range args {
		if isHelp(arg) {
			return true
		}
	}
	return false
}

func parseNoArgs(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return usageError{err: err}
	}
	if fs.NArg() > 0 {
		return usageError{err: fmt.Errorf("unexpected argument %q", fs.Arg(0))}
	}
	return nil
}
```

Update `runConfigure`, `runMigrate`, and `runMigratePreview`:

```go
if hasHelpArg(args) {
	printConfigureUsage(out)
	return nil
}
fs := flag.NewFlagSet("configure", flag.ContinueOnError)
fs.SetOutput(errOut)
if err := parseNoArgs(fs, args); err != nil {
	return err
}
```

For `migrate`, keep `workspace := fs.String(...)`, then call `parseNoArgs`. For `migrate-preview`, same pattern with `printMigratePreviewUsage`.

- [x] **Step 5: Run tests and commit**

Run:

```bash
env GOPATH=/tmp/go GOMODCACHE=/tmp/go/pkg/mod GOCACHE=/tmp/go-build go test ./cmd/bkt2gh
```

Expected: PASS.

Commit:

```bash
git add cmd/bkt2gh/main.go cmd/bkt2gh/main_test.go
git commit -m "fix: reject unexpected cli arguments"
```

### Task 2: Remove dead dry-run migrator test/code

**Files:**
- Modify: `internal/gitops/gitops.go`
- Modify: `internal/gitops/gitops_test.go`

- [x] **Step 1: Confirm `DryRunMigrator` is unused**

Run:

```bash
rg -n "DryRunMigrator" .
```

Expected before edit: matches only `internal/gitops/gitops.go` and `internal/gitops/gitops_test.go`.

- [x] **Step 2: Delete meaningless test and unused type**

Remove from `internal/gitops/gitops_test.go`:

```go
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
```

Remove from `internal/gitops/gitops.go`:

```go
type DryRunMigrator struct {
	Out io.Writer
}

func (m DryRunMigrator) Migrate(ctx context.Context, repo model.Repository, githubCloneURL string) error {
	if m.Out != nil {
		fmt.Fprintf(m.Out, "DRY-RUN git: clone --mirror %s; push --mirror %s\n", repo.CloneURL, githubCloneURL)
	}
	return nil
}
```

- [x] **Step 3: Run package tests and usage search**

Run:

```bash
rg -n "DryRunMigrator" .
env GOPATH=/tmp/go GOMODCACHE=/tmp/go/pkg/mod GOCACHE=/tmp/go-build go test ./internal/gitops
```

Expected: `rg` has no matches. `go test` PASS.

- [x] **Step 4: Commit**

```bash
git add internal/gitops/gitops.go internal/gitops/gitops_test.go
git commit -m "test: remove unused dry run migrator"
```

### Task 3: Per-repository cleanup in migration runner

**Files:**
- Modify: `internal/migrate/migrate.go`
- Modify: `internal/migrate/migrate_test.go`

- [x] **Step 1: Add failing cleanup timing and failure-path tests**

Extend `fakeGit` and `fakePreparedMirror` in `internal/migrate/migrate_test.go`:

```go
type fakeGit struct {
	called     bool
	prepareErr error
	prepared   *fakePreparedMirror
	preparedAll []*fakePreparedMirror
	onPrepare  func()
	nextPushErr error
}

func (f *fakeGit) Prepare(ctx context.Context, repo model.Repository) (interface {
	Push(ctx context.Context, githubCloneURL string) error
	Cleanup() error
}, error) {
	f.called = true
	if f.onPrepare != nil {
		f.onPrepare()
	}
	if f.prepareErr != nil {
		return nil, f.prepareErr
	}
	f.prepared = &fakePreparedMirror{pushErr: f.nextPushErr}
	f.preparedAll = append(f.preparedAll, f.prepared)
	return f.prepared, nil
}

type fakePreparedMirror struct {
	dst       string
	pushErr   error
	cleanedUp bool
}

func (f *fakePreparedMirror) Push(ctx context.Context, githubCloneURL string) error {
	f.dst = githubCloneURL
	return f.pushErr
}

func (f *fakePreparedMirror) Cleanup() error {
	f.cleanedUp = true
	return nil
}
```

Add tests:

```go
func TestRunnerCleansUpEachRepositoryBeforePreparingNext(t *testing.T) {
	bb := &fakeBitbucket{repos: []model.Repository{
		{Name: "Repo One", Slug: "repo-one", Private: true, CloneURL: "bb-url-1"},
		{Name: "Repo Two", Slug: "repo-two", Private: true, CloneURL: "bb-url-2"},
	}}
	gh := &fakeGitHub{cloneURL: "gh-url"}
	git := &fakeGit{}
	git.onPrepare = func() {
		if len(git.preparedAll) == 1 && !git.preparedAll[0].cleanedUp {
			t.Fatal("first repository was not cleaned up before preparing second repository")
		}
	}
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
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if len(git.preparedAll) != 2 || !git.preparedAll[0].cleanedUp || !git.preparedAll[1].cleanedUp {
		t.Fatalf("cleanup state = %+v", git.preparedAll)
	}
}

func TestRunnerCleansUpAfterPushFailure(t *testing.T) {
	bb := &fakeBitbucket{repos: []model.Repository{{Name: "Repo One", Slug: "repo-one", CloneURL: "bb-url"}}}
	gh := &fakeGitHub{cloneURL: "gh-url"}
	git := &fakeGit{nextPushErr: errors.New("push failed")}
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
	if len(results) != 1 || results[0].Status != model.StatusFailed {
		t.Fatalf("results = %+v, want one failed result", results)
	}
	if len(git.preparedAll) != 1 || !git.preparedAll[0].cleanedUp {
		t.Fatalf("cleanup after push failure = %+v", git.preparedAll)
	}
}
```

- [x] **Step 2: Run targeted tests and verify failure**

Run:

```bash
env GOPATH=/tmp/go GOMODCACHE=/tmp/go/pkg/mod GOCACHE=/tmp/go-build go test ./internal/migrate -run 'TestRunnerCleansUpEachRepositoryBeforePreparingNext|TestRunnerCleansUpAfterPushFailure'
```

Expected: first test FAIL before implementation because cleanup uses `defer` in loop and runs only when `Runner.Run` returns.

- [x] **Step 3: Implement repo-scoped migration helper**

In `internal/migrate/migrate.go`, replace the body of the non-dry-run loop with:

```go
results := []model.RepoResult{}
for _, repo := range selected {
	results = append(results, r.migrateOne(ctx, repo, visibilityPolicy))
}
PrintSummary(out, results)
return results, nil
```

Add helper:

```go
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
```

- [x] **Step 4: Run package tests and commit**

```bash
env GOPATH=/tmp/go GOMODCACHE=/tmp/go/pkg/mod GOCACHE=/tmp/go-build go test ./internal/migrate
git add internal/migrate/migrate.go internal/migrate/migrate_test.go
git commit -m "fix: cleanup mirrors after each repository"
```

### Task 4: Bitbucket response body close per page

**Files:**
- Modify: `internal/bitbucket/client.go`
- Modify: `internal/bitbucket/client_test.go`

- [x] **Step 1: Add failing test for response body close count**

Add to `internal/bitbucket/client_test.go`:

```go
type closeCountingTransport struct {
	bodies []*closeCountingBody
}

func (t *closeCountingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body := &closeCountingBody{Reader: strings.NewReader(`{"values":[]}`)}
	t.bodies = append(t.bodies, body)
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       body,
		Request:    req,
	}, nil
}

type closeCountingBody struct {
	*strings.Reader
	closed bool
}

func (b *closeCountingBody) Close() error {
	b.closed = true
	return nil
}

func TestListRepositoriesClosesResponseBodyBeforeReturn(t *testing.T) {
	transport := &closeCountingTransport{}
	client := NewClient("user", "pass")
	client.BaseURL = "https://bitbucket.example.test"
	client.HTTPClient = &http.Client{Transport: transport}

	_, err := client.ListRepositories(context.Background(), "team")
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}
	if len(transport.bodies) != 1 {
		t.Fatalf("body count = %d, want 1", len(transport.bodies))
	}
	if !transport.bodies[0].closed {
		t.Fatal("response body was not closed")
	}
}
```

- [x] **Step 2: Implement immediate close**

In `internal/bitbucket/client.go`, replace `defer resp.Body.Close()` with explicit close after decode:

```go
var page repositoriesPage
decodeErr := json.NewDecoder(resp.Body).Decode(&page)
closeErr := resp.Body.Close()
if decodeErr != nil {
	return nil, fmt.Errorf("failed to decode Bitbucket repositories response: %w", decodeErr)
}
if closeErr != nil {
	return nil, fmt.Errorf("failed to close Bitbucket repositories response: %w", closeErr)
}
```

For non-2xx status, read a small body if desired, then close before returning. Minimum required:

```go
if resp.StatusCode < 200 || resp.StatusCode > 299 {
	resp.Body.Close()
	return nil, fmt.Errorf("Bitbucket API returned %s while listing repositories", resp.Status)
}
```

- [x] **Step 3: Remove mutable global test URL**

In `TestListRepositoriesPaginates`, avoid `serverURLPlaceholder`. Declare:

```go
var server *httptest.Server
server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// use server.URL inside handler
}))
```

Then remove:

```go
var serverURLPlaceholder = "http://example.test"
```

- [x] **Step 4: Run package tests and commit**

```bash
env GOPATH=/tmp/go GOMODCACHE=/tmp/go/pkg/mod GOCACHE=/tmp/go-build go test ./internal/bitbucket
git add internal/bitbucket/client.go internal/bitbucket/client_test.go
git commit -m "fix: close bitbucket responses per page"
```

### Task 5: Final verification

**Files:**
- Verify only.

- [x] **Step 1: Run full tests**

```bash
env GOPATH=/tmp/go GOMODCACHE=/tmp/go/pkg/mod GOCACHE=/tmp/go-build go test ./...
```

Expected: all packages PASS.

- [x] **Step 2: Run vet**

```bash
env GOPATH=/tmp/go GOMODCACHE=/tmp/go/pkg/mod GOCACHE=/tmp/go-build go vet ./...
```

Expected: exit 0, no output.

- [x] **Step 3: Run focused searches**

```bash
rg -n "DryRunMigrator|serverURLPlaceholder" cmd internal
```

Expected: no matches.

```bash
rg -n "defer prepared.Cleanup\\(\\)|defer resp.Body.Close\\(\\)" internal/migrate internal/bitbucket
```

Expected: no matches.

- [x] **Step 4: Commit verification note only if files changed**

If verification caused no file changes, do not commit. If formatting changed files, commit:

```bash
git status --short
git add <changed-files>
git commit -m "chore: apply final formatting"
```

---

## Self Review

- Spec coverage: CLI unexpected args/help consistency, meaningless always-pass test removal, per-repo cleanup, response body close, full verification covered.
- Placeholder scan: no TBD/TODO/fill-later items.
- Type consistency: plan uses existing `Runner`, `model.RepoResult`, `policy.VisibilityPolicy`, `fakeGit`, `fakePreparedMirror`, `Bitbucket Client` names.
