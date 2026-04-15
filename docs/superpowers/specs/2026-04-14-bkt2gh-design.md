# bkt2gh Design

## Goal

Build a single Go CLI that migrates selected Bitbucket Cloud repositories to GitHub with clear prompts, dry-run support, and maintainable package boundaries.

## Architecture

The CLI entrypoint parses `bkt2gh migrate` flags and delegates to `internal/migrate`. Config loading is isolated in `internal/config`; Bitbucket and GitHub REST calls live in separate client packages; terminal prompts stay in `internal/prompt`; git subprocess work stays in `internal/gitops`.

Dry-run mode calls Bitbucket and GitHub read/check APIs, lets the user select repositories, checks GitHub repository availability and owner access, then prints the migration plan. It does not create GitHub repositories and does not run git clone or push.

## Components

- `cmd/bkt2gh`: main program.
- `internal/config`: encrypted config loading, interactive config creation, validation.
- `internal/model`: shared repository and result types.
- `internal/policy`: visibility policy enum and decision function.
- `internal/bitbucket`: Bitbucket Cloud repository listing with pagination.
- `internal/github`: GitHub repository creation and already-exists skip handling.
- `internal/gitops`: mirror clone, optional LFS fetch/push, mirror push.
- `internal/prompt`: numeric multi-select and visibility policy prompt.
- `internal/migrate`: orchestration and summary output.

## Testing

Unit tests cover visibility logic, env parsing, Bitbucket pagination, GitHub create/skip/check behavior, dry-run preflight behavior, and git command construction through fake runners.
