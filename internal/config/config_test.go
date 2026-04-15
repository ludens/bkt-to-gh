package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type memoryKeyring struct {
	values map[string]string
}

func newMemoryKeyring() *memoryKeyring {
	return &memoryKeyring{values: map[string]string{}}
}

func (m *memoryKeyring) Get(service, user string) (string, error) {
	value, ok := m.values[service+"/"+user]
	if !ok {
		return "", ErrKeyNotFound
	}
	return value, nil
}

func (m *memoryKeyring) Set(service, user, password string) error {
	if m.values == nil {
		m.values = map[string]string{}
	}
	m.values[service+"/"+user] = password
	return nil
}

func TestDefaultPathUsesUserConfigDir(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("XDG_CONFIG_HOME is Unix-specific")
	}
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath() error = %v", err)
	}

	want := filepath.Join(dir, "bkt2gh", "config.yaml")
	if path != want {
		t.Fatalf("DefaultPath() = %q, want %q", path, want)
	}
}

func TestWriteAndLoadEncryptedConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	keyring := newMemoryKeyring()
	want := Config{
		BitbucketUsername:    "person@example.com",
		BitbucketAppPassword: "bb-pass",
		BitbucketWorkspace:   "team",
		GitHubToken:          "gh-token",
		GitHubOwner:          "acme",
	}

	if err := Write(path, want, keyring); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"person@example.com", "bb-pass", "team", "gh-token", "acme"} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("encrypted config leaked %q in:\n%s", secret, string(raw))
		}
	}

	got, err := Load(path, keyring)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got != want {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}

func TestLoadAllowsEnvironmentOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	keyring := newMemoryKeyring()
	if err := Write(path, Config{BitbucketWorkspace: "from-file"}, keyring); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	t.Setenv("BITBUCKET_WORKSPACE", "from-env")

	cfg, err := Load(path, keyring)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.BitbucketWorkspace != "from-env" {
		t.Fatalf("BitbucketWorkspace = %q, want from-env", cfg.BitbucketWorkspace)
	}
}

func TestLoadMissingFileUsesEnvironmentOnly(t *testing.T) {
	t.Setenv("BITBUCKET_USERNAME", "bb-user")
	t.Setenv("BITBUCKET_APP_PASSWORD", "bb-pass")
	t.Setenv("BITBUCKET_WORKSPACE", "team")
	t.Setenv("GITHUB_TOKEN", "gh-token")
	t.Setenv("GITHUB_OWNER", "acme")

	cfg, err := Load(filepath.Join(t.TempDir(), "config.yaml"), newMemoryKeyring())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestLoadReportsMissingKeyringEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writerKeyring := newMemoryKeyring()
	if err := Write(path, Config{BitbucketWorkspace: "team"}, writerKeyring); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	_, err := Load(path, newMemoryKeyring())
	if !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("Load() error = %v, want ErrKeyNotFound", err)
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

func TestConfigureInteractiveWritesEncryptedConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	input := strings.NewReader(strings.Join([]string{
		"person@example.com",
		"bb-app-password",
		"team",
		"gh-token",
		"acme",
		"",
	}, "\n"))
	output := new(strings.Builder)
	keyring := newMemoryKeyring()

	_, err := ConfigureInteractive(input, output, path, keyring)
	if err != nil {
		t.Fatalf("ConfigureInteractive() error = %v", err)
	}

	cfg, err := Load(path, keyring)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.BitbucketUsername != "person@example.com" {
		t.Fatalf("BitbucketUsername = %q", cfg.BitbucketUsername)
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
		"Wrote encrypted config",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q in:\n%s", want, text)
		}
	}
}

func TestConfigureInteractiveIfAllowedDoesNotOverwriteWhenDeclined(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	keyring := newMemoryKeyring()
	original := Config{BitbucketUsername: "old@example.com", BitbucketWorkspace: "old-team"}
	if err := Write(path, original, keyring); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	output := new(strings.Builder)
	_, configured, err := ConfigureInteractiveIfAllowed(strings.NewReader("n\n"), output, path, keyring)
	if err != nil {
		t.Fatalf("ConfigureInteractiveIfAllowed() error = %v", err)
	}
	if configured {
		t.Fatal("configured = true, want false")
	}
	got, err := Load(path, keyring)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.BitbucketUsername != original.BitbucketUsername {
		t.Fatalf("config overwritten = %#v, want %#v", got, original)
	}
	if !strings.Contains(output.String(), "config.yaml already exists") {
		t.Fatalf("output missing overwrite prompt:\n%s", output.String())
	}
}
