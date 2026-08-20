# AGENTS.md

## Landmines

- `config.yaml` is NOT editable YAML. Extension lies: it is an AES-GCM encrypted envelope, key stored in the OS keychain (service `bkt2gh`, account `config`). Never edit/parse by hand. Change config only via `bkt2gh configure` or env vars.
- Env vars alone do NOT enable headless/CI runs. `migrate` / `migrate-preview` force interactive `configure` whenever the config file does not exist — even when all 5 `BITBUCKET_*` / `GITHUB_*` vars are set. Create the config once on a machine with a working OS keychain (macOS Keychain; Linux secret-service/dbus — go-keyring fails on headless Linux).

## Release

- Any `v*` tag push runs GoReleaser: builds, creates a GitHub release, AND publishes/updates the Homebrew cask in the separate `ludens/homebrew-tap` repo. Requires `HOMEBREW_TAP_GITHUB_TOKEN` secret. Don't tag casually.

## Non-discoverable commands

- Regenerate third-party license texts after dependency changes:
  `go run github.com/google/go-licenses@latest save ./... --save_path=third_party_licenses`

## Docs

- README.md (English) and README.ko.md (Korean) are kept in sync — update both.
