# GoReleaser Snapshot Design

## Goal

Add the first release packaging layer for `bkt2gh` using GoReleaser. This step prepares local and CI-friendly release artifacts, but does not publish Homebrew, winget, or GitHub Actions automation yet.

## Scope

Create a root `.goreleaser.yaml` that builds the existing CLI entrypoint at `./cmd/bkt2gh`.

The release config will produce:

- Linux, macOS, and Windows binaries.
- `amd64` and `arm64` builds.
- compressed archives with platform-specific names.
- a checksum file for release verification.

This step excludes:

- Homebrew tap formula generation.
- winget manifests.
- GitHub Actions workflow.
- module path rename.
- CLI behavior changes.

## Design

Use GoReleaser's standard OSS config format with schema metadata for editor support. Keep the config minimal so it is easy to validate with `goreleaser check` and easy to extend later.

Builds will target `main: ./cmd/bkt2gh`, binary name `bkt2gh`, `CGO_ENABLED=0`, and common OS/architecture combinations. Windows archives will use `.zip`; other platforms will use `.tar.gz`.

## Validation

Run:

```bash
goreleaser check
goreleaser release --snapshot --clean
```

If GoReleaser is unavailable locally, validate the YAML structure by review and leave exact command output as a follow-up.
