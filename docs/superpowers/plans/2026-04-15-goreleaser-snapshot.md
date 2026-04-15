# GoReleaser Snapshot Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a minimal GoReleaser config that builds `bkt2gh` release artifacts for Linux, macOS, and Windows.

**Architecture:** Keep release packaging isolated in root `.goreleaser.yaml`. The CLI source and runtime behavior remain unchanged.

**Tech Stack:** Go 1.22, GoReleaser OSS config, GitHub Release-compatible archives and checksums.

---

### Task 1: Add GoReleaser Config

**Files:**
- Create: `.goreleaser.yaml`

- [ ] **Step 1: Create `.goreleaser.yaml`**

```yaml
# yaml-language-server: $schema=https://goreleaser.com/static/schema.json
version: 2

project_name: bkt2gh

before:
  hooks:
    - go mod tidy

builds:
  - id: bkt2gh
    main: ./cmd/bkt2gh
    binary: bkt2gh
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64

archives:
  - id: bkt2gh
    ids:
      - bkt2gh
    name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- .Os }}_
      {{- .Arch }}
    formats:
      - tar.gz
    format_overrides:
      - goos: windows
        formats:
          - zip

checksum:
  name_template: checksums.txt

snapshot:
  version_template: "{{ incpatch .Version }}-next"

changelog:
  use: git
  sort: asc
```

- [ ] **Step 2: Validate config if GoReleaser is installed**

Run:

```bash
goreleaser check
```

Expected: config valid. If `goreleaser` is not installed, record that validation was skipped.

- [ ] **Step 3: Run snapshot build if GoReleaser is installed**

Run:

```bash
goreleaser release --snapshot --clean
```

Expected: `dist/` contains platform archives and `checksums.txt`. If `goreleaser` is not installed, record that snapshot build was skipped.

- [ ] **Step 4: Run Go tests**

Run:

```bash
go test ./...
```

Expected: all packages pass.
