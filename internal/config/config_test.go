package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadReadsDotEnvAndEnvironment(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte(strings.Join([]string{
		"BITBUCKET_USERNAME=bb-user",
		"BITBUCKET_APP_PASSWORD=bb-pass",
		"BITBUCKET_WORKSPACE=bb-workspace",
		"GITHUB_TOKEN=gh-token",
		"GITHUB_OWNER=gh-owner",
		"",
	}, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.BitbucketUsername != "bb-user" {
		t.Fatalf("BitbucketUsername = %q", cfg.BitbucketUsername)
	}
	if cfg.BitbucketAppPassword != "bb-pass" {
		t.Fatalf("BitbucketAppPassword = %q", cfg.BitbucketAppPassword)
	}
	if cfg.BitbucketWorkspace != "bb-workspace" {
		t.Fatalf("BitbucketWorkspace = %q", cfg.BitbucketWorkspace)
	}
	if cfg.GitHubToken != "gh-token" {
		t.Fatalf("GitHubToken = %q", cfg.GitHubToken)
	}
	if cfg.GitHubOwner != "gh-owner" {
		t.Fatalf("GitHubOwner = %q", cfg.GitHubOwner)
	}
}

func TestLoadAllowsEnvironmentOverride(t *testing.T) {
	t.Setenv("BITBUCKET_WORKSPACE", "from-env")

	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("BITBUCKET_WORKSPACE=from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.BitbucketWorkspace != "from-env" {
		t.Fatalf("BitbucketWorkspace = %q, want from-env", cfg.BitbucketWorkspace)
	}
}

func TestValidateReportsMissingFields(t *testing.T) {
	err := (Config{GitHubToken: "gh-token"}).Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want missing fields")
	}
	if !strings.Contains(err.Error(), "BITBUCKET_USERNAME") {
		t.Fatalf("Validate() error = %q, want BITBUCKET_USERNAME", err)
	}
	if !strings.Contains(err.Error(), "GITHUB_OWNER") {
		t.Fatalf("Validate() error = %q, want GITHUB_OWNER", err)
	}
}

func TestWriteDotEnvCreatesExpectedFile(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		BitbucketUsername:    "bb-user",
		BitbucketAppPassword: "bb-pass",
		BitbucketWorkspace:   "bb-workspace",
		GitHubToken:          "gh-token",
		GitHubOwner:          "gh-owner",
	}

	if err := WriteDotEnv(filepath.Join(dir, ".env"), cfg); err != nil {
		t.Fatalf("WriteDotEnv() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	for _, want := range []string{
		"BITBUCKET_USERNAME=bb-user",
		"BITBUCKET_APP_PASSWORD=bb-pass",
		"BITBUCKET_WORKSPACE=bb-workspace",
		"GITHUB_TOKEN=gh-token",
		"GITHUB_OWNER=gh-owner",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf(".env missing %q in:\n%s", want, text)
		}
	}
}

func TestConfigureInteractiveShowsTokenScopeGuidanceAndEmailLabel(t *testing.T) {
	dir := t.TempDir()
	input := strings.NewReader(strings.Join([]string{
		"person@example.com",
		"bb-app-password",
		"team",
		"gh-token",
		"acme",
		"",
	}, "\n"))
	output := new(strings.Builder)

	_, err := ConfigureInteractive(input, output, filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatalf("ConfigureInteractive() error = %v", err)
	}

	text := output.String()
	for _, want := range []string{
		"Bitbucket username (email)",
		"Bitbucket app password permissions",
		"Bitbucket app password (hidden)",
		"Repositories: Read",
		"GitHub token permissions",
		"GitHub token (hidden)",
		"Administration: Read and write",
		"Contents: Read and write",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q in:\n%s", want, text)
		}
	}
}

func TestConfigureInteractiveIfAllowedDoesNotOverwriteWhenDeclined(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	original := "BITBUCKET_USERNAME=old@example.com\n"
	if err := os.WriteFile(envPath, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	output := new(strings.Builder)
	_, configured, err := ConfigureInteractiveIfAllowed(strings.NewReader("n\n"), output, envPath)
	if err != nil {
		t.Fatalf("ConfigureInteractiveIfAllowed() error = %v", err)
	}
	if configured {
		t.Fatal("configured = true, want false")
	}
	got, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Fatalf(".env overwritten = %q, want %q", string(got), original)
	}
	if !strings.Contains(output.String(), ".env already exists") {
		t.Fatalf("output missing overwrite prompt:\n%s", output.String())
	}
}

func TestConfigureInteractiveIfAllowedOverwritesWhenConfirmed(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("BITBUCKET_USERNAME=old@example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	input := strings.NewReader(strings.Join([]string{
		"y",
		"person@example.com",
		"bb-app-password",
		"team",
		"gh-token",
		"acme",
		"",
	}, "\n"))

	_, configured, err := ConfigureInteractiveIfAllowed(input, new(strings.Builder), envPath)
	if err != nil {
		t.Fatalf("ConfigureInteractiveIfAllowed() error = %v", err)
	}
	if !configured {
		t.Fatal("configured = false, want true")
	}
	got, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "BITBUCKET_USERNAME=person@example.com") {
		t.Fatalf(".env not overwritten with new config:\n%s", string(got))
	}
}
