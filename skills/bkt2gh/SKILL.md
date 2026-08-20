---
name: bkt2gh
description: Use when running or scripting the bkt2gh CLI to migrate Bitbucket Cloud repositories to GitHub, or when troubleshooting bkt2gh configuration, migration preview, LFS handling, or mirror-push behavior.
---

# bkt2gh

Go CLI that migrates selected Bitbucket Cloud repositories to GitHub using `git clone --mirror` + `git push --mirror` (branches and tags included).

## Commands

| Command | Purpose |
| --- | --- |
| `bkt2gh configure` | Create/update encrypted config interactively |
| `bkt2gh migrate-preview [--workspace name]` | Preflight: checks Bitbucket/GitHub access and target repo availability; writes nothing |
| `bkt2gh migrate [--workspace name]` | Migrate selected repositories |

## Configuration

- `bkt2gh configure` writes an encrypted `config.yaml` to the OS user config dir; the key is stored in the OS keychain.
- Env vars override config values: `BITBUCKET_USERNAME`, `BITBUCKET_APP_PASSWORD`, `BITBUCKET_WORKSPACE`, `GITHUB_TOKEN`, `GITHUB_OWNER`.

Required Bitbucket app password permissions (all Read): Account, Workspace membership, Projects, Repositories.
Required GitHub token permissions: Metadata Read-only; Administration Read+write; Contents Read+write.

## Landmines

- `config.yaml` is an AES-GCM encrypted envelope, not editable YAML. Change config only via `configure` or env vars.
- A config file must exist before `migrate` / `migrate-preview` run; env vars alone do not skip interactive setup. Create the config once on a machine with a working OS keychain (go-keyring fails on headless Linux).
- `--workspace` applies to that single invocation only; it never persists.
- Existing GitHub repos are skipped, never overwritten. Target repo names = Bitbucket slugs.
- `git-lfs` is optional; LFS fetch/push is silently skipped when `git-lfs` is absent.

## Repository Selection (TUI)

- `N` select/deselect · `1,3` multi-select · `all` / `none` · `filter text` · `done`

## Visibility Policy

`all-private` · `all-public` · `follow-source`

## Migration Order

mirror clone → LFS fetch (optional) → create GitHub repo → `git remote set-url origin` → LFS push (optional) → `git push --mirror origin` → cleanup temp dir.
