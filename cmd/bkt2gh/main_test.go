package main

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRunRootHelpReturnsNil(t *testing.T) {
	for _, args := range [][]string{
		{"--help"},
		{"-h"},
		{"help"},
	} {
		t.Run(args[0], func(t *testing.T) {
			if err := run(args); err != nil {
				t.Fatalf("run(%v) error = %v, want nil", args, err)
			}
		})
	}
}

func TestRunCommandHelpReturnsNil(t *testing.T) {
	for _, args := range [][]string{
		{"migrate", "--help"},
		{"configure", "--help"},
	} {
		t.Run(args[0], func(t *testing.T) {
			if err := run(args); err != nil {
				t.Fatalf("run(%v) error = %v, want nil", args, err)
			}
		})
	}
}

func TestMigrateConfigureFlagIsRemoved(t *testing.T) {
	if err := run([]string{"migrate", "--configure"}); err == nil {
		t.Fatal("run([migrate --configure]) error = nil, want error")
	}
}

func TestRunCLIInvalidCommandReturnsUsageExitCode(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder

	code := runCLI(context.Background(), strings.NewReader(""), &stdout, &stderr, []string{"wat"})

	if code != 2 {
		t.Fatalf("runCLI() code = %d, want 2", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), `Error: unknown command "wat"`) {
		t.Fatalf("stderr missing unknown command error: %q", stderr.String())
	}
}

func TestRunCLIMigrateFlagErrorUsesStderrOnly(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder

	code := runCLI(context.Background(), strings.NewReader(""), &stdout, &stderr, []string{"migrate", "--definitely-not-a-flag"})

	if code != 2 {
		t.Fatalf("runCLI() code = %d, want 2", code)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("stderr missing flag parse error: %q", stderr.String())
	}
}

func TestRunMigrateUsesProvidedContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runWithIO(ctx, strings.NewReader(""), new(strings.Builder), new(strings.Builder), []string{"migrate", "--workspace", "team"})

	if err == nil {
		t.Fatal("runWithIO() error = nil, want context cancellation or configuration error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runWithIO() error = %v, want context cancellation before migration work", err)
	}
}
