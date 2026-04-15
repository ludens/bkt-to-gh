# bkt2gh

Go CLI for migrating Bitbucket Cloud repositories to GitHub.

[한국어](README.ko.md)

It creates new GitHub repositories for selected Bitbucket repositories and migrates Git history, including branches and tags, using `git clone --mirror` and `git push --mirror`. Before running a real migration, dry-run mode checks target repositories, GitHub repository creation availability, and repository visibility policy.

## Features

- List repositories in a Bitbucket Cloud workspace
- Select repositories to migrate from the terminal
- Create GitHub repositories
- Migrate Git history with mirror clone/push
- Attempt LFS object fetch/push when `git-lfs` is installed
- Choose GitHub repository visibility policy
- Run dry-run preflight checks
- Load configuration from `.env` with environment variable overrides

## Requirements

- Go 1.22 or later
- Git
- Bitbucket Cloud account and app password
- GitHub token
- Optional: `git-lfs` for migrating Git LFS repositories

## Installation

Run from source:

```bash
go run ./cmd/bkt2gh --help
```

Build a binary:

```bash
go build -o bkt2gh ./cmd/bkt2gh
./bkt2gh --help
```

## Quick Start

1. Create the configuration file:

```bash
./bkt2gh configure
```

2. Review the migration plan with dry-run:

```bash
./bkt2gh migrate --dry-run
```

3. Run the migration:

```bash
./bkt2gh migrate
```

Temporarily use another Bitbucket workspace:

```bash
./bkt2gh migrate --workspace my-workspace --dry-run
```

## Configuration

`bkt2gh configure` creates a `.env` file in the current directory. If `.env` already exists, it asks whether to overwrite it.

Required values:

```dotenv
BITBUCKET_USERNAME=you@example.com
BITBUCKET_APP_PASSWORD=your-bitbucket-app-password
BITBUCKET_WORKSPACE=your-workspace
GITHUB_TOKEN=your-github-token
GITHUB_OWNER=your-github-user-or-org
```

Configuration priority:

1. Environment variables
2. `.env`

If a value exists in `.env` and an environment variable with the same name is also set, the environment variable is used.

## Token Permissions

Bitbucket app password permissions:

- Account: Read
- Workspace membership: Read
- Projects: Read
- Repositories: Read

GitHub token permissions:

- Metadata: Read-only
- Administration: Read and write
- Contents: Read and write

When using a GitHub fine-grained token, it must be able to create repositories for the user or organization referenced by `GITHUB_OWNER`.

## Usage

```text
Usage:
  bkt2gh configure
  bkt2gh migrate [--workspace name] [--dry-run]

Commands:
  configure  create or update .env interactively
  migrate    migrate selected Bitbucket repositories to GitHub

Flags:
  -h, --help show help
```

### `configure`

Create or update `.env` interactively.

```bash
./bkt2gh configure
```

### `migrate`

List Bitbucket repositories and migrate the repositories selected by the user to GitHub.

```bash
./bkt2gh migrate
```

Options:

- `--workspace name`: workspace to use instead of `BITBUCKET_WORKSPACE` from `.env`
- `--dry-run`: print only the migration plan without creating GitHub repositories or running Git clone/push

## Repository Selection

When `migrate` runs, it opens a repository selection screen.

Commands:

- Number: select/deselect that repository
- `1,3`: select/deselect multiple repositories
- `all`: select all currently visible repositories
- `none`: deselect all currently visible repositories
- `filter text`: filter by name or slug
- `done`: finish selection

## Visibility Policy

After selecting repositories, choose the GitHub repository visibility policy.

- `all-private`: create all GitHub repositories as private
- `all-public`: create all GitHub repositories as public
- `follow-source`: follow the public/private state of the Bitbucket repository

## dry-run

Dry-run calls the Bitbucket/GitHub APIs to verify the plan, but it does not create repositories or run Git commands.

```bash
./bkt2gh migrate --dry-run
```

Checked items:

- Whether the Bitbucket repository list can be loaded
- Whether the GitHub token and owner are accessible
- Whether target GitHub repository names are available
- Result of applying the visibility policy

## Real Migration Behavior

When run without dry-run, each selected repository is processed in this order:

1. Clone the Bitbucket repository into a temporary directory with `git clone --mirror`
2. Attempt `git lfs fetch --all` when `git-lfs` is available
3. Create the GitHub repository
4. Change `origin` to the GitHub clone URL
5. Attempt `git lfs push --all origin` when `git-lfs` is available
6. Run `git push --mirror origin`
7. Clean up the temporary directory

If a GitHub repository with the same name already exists, it is skipped without overwriting.

## Development

Test:

```bash
go test ./...
```

Build:

```bash
go build -o bkt2gh ./cmd/bkt2gh
```

## Notes

- Existing target GitHub repositories are not overwritten.
- GitHub repository names use Bitbucket repository slugs.
- Do not commit `.env` because it contains sensitive information.
- If `git-lfs` is not installed, LFS handling is skipped and only the regular Git mirror push is performed.
