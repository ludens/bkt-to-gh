# Encrypted Config Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Store primary configuration in encrypted OS user config `config.yaml`.

**Architecture:** `internal/config` owns path resolution, keyring access, encryption, and load/write. `cmd/bkt2gh` asks `internal/config` for the default path and keyring, while tests inject an in-memory keyring.

**Tech Stack:** Go stdlib AES-GCM/JSON/file APIs, `github.com/zalando/go-keyring`, existing stdlib CLI flag handling.

---

### Task 1: Config Storage and Encryption

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [x] Add tests for OS config path resolution, encrypted write/load, no plaintext leakage, env var override, missing config with env vars, and missing keyring key.
- [x] Implement `DefaultPath`, `Keyring`, `DefaultKeyring`, `Load`, and `Write`.
- [x] Store only an encrypted YAML envelope on disk.
- [x] Keep environment variable override behavior.

### Task 2: CLI Integration

**Files:**
- Modify: `cmd/bkt2gh/main.go`
- Modify: `cmd/bkt2gh/main_test.go`

- [x] Add tests for `configure` writing encrypted config.
- [x] Route `configure`, `migrate-preview`, and `migrate` through OS config path and keyring.
- [x] Create encrypted config when config is missing.
- [x] Update help text to encrypted `config.yaml`.

### Task 3: Documentation and Verification

**Files:**
- Modify: `README.md`
- Modify: `README.ko.md`
- Modify: `go.mod`
- Modify: `go.sum`

- [x] Document OS config paths, encrypted storage, and env var override.
- [x] Add `github.com/zalando/go-keyring`.
- [x] Run `go mod tidy`.
- [x] Run `go test ./...`.
- [x] Run `go build -o /tmp/bkt2gh ./cmd/bkt2gh`.
