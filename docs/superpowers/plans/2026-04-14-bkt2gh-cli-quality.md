# bkt2gh CLI Quality Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Improve `bkt2gh` CLI behavior without changing migration semantics.

**Architecture:** Keep the existing stdlib `flag` command structure. Add a thin CLI boundary that owns context, streams, and exit-code mapping while keeping migration orchestration in `internal/migrate`.

**Tech Stack:** Go 1.22, standard library `flag`, `context`, `os/signal`, existing package tests.

---

### Task 1: CLI Boundary

**Files:**
- Modify: `cmd/bkt2gh/main.go`
- Test: `cmd/bkt2gh/main_test.go`

- [x] Add tests for `runCLI` returning exit code `2` for invalid usage.
- [x] Add tests proving flag parse errors write diagnostics to stderr, not stdout.
- [x] Add tests proving `migrate` receives the caller-provided context.
- [x] Implement injectable streams and context-aware command execution.
- [x] Add `signal.NotifyContext` in `main`.
- [x] Run `env GOPATH=/tmp/go GOMODCACHE=/tmp/go/pkg/mod GOCACHE=/tmp/go-build go test ./...`.
