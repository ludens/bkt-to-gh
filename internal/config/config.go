package config

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zalando/go-keyring"
	"golang.org/x/term"
)

const (
	appName          = "bkt2gh"
	configFileName   = "config.yaml"
	keyringService   = "bkt2gh"
	keyringAccount   = "config"
	encryptionMethod = "os-keyring-aes-gcm"
)

var ErrKeyNotFound = keyring.ErrNotFound

type Keyring interface {
	Get(service, user string) (string, error)
	Set(service, user, password string) error
}

type OSKeyring struct{}

func (OSKeyring) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (OSKeyring) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func DefaultKeyring() Keyring {
	return OSKeyring{}
}

type Config struct {
	BitbucketUsername    string
	BitbucketAppPassword string
	BitbucketWorkspace   string
	GitHubToken          string
	GitHubOwner          string
}

type encryptedFile struct {
	Version    int
	Encryption string
	Nonce      string
	Data       string
}

type fileConfig struct {
	BitbucketUsername    string `json:"bitbucket_username"`
	BitbucketAppPassword string `json:"bitbucket_app_password"`
	BitbucketWorkspace   string `json:"bitbucket_workspace"`
	GitHubToken          string `json:"github_token"`
	GitHubOwner          string `json:"github_owner"`
}

func DefaultPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, appName, configFileName), nil
}

func Load(path string, kr Keyring) (Config, error) {
	cfg := Config{}
	if file, err := os.Open(path); err == nil {
		defer file.Close()
		loaded, err := readEncrypted(file, kr)
		if err != nil {
			return Config{}, fmt.Errorf("failed to read %s: %w", path, err)
		}
		cfg = loaded
	} else if !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("failed to read %s: %w", path, err)
	}
	applyEnvOverrides(&cfg)
	return cfg, nil
}

func Write(path string, cfg Config, kr Keyring) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	encoded, err := encodeEncrypted(cfg, kr)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

func (c Config) Validate() error {
	missing := []string{}
	if strings.TrimSpace(c.BitbucketUsername) == "" {
		missing = append(missing, "BITBUCKET_USERNAME")
	}
	if strings.TrimSpace(c.BitbucketAppPassword) == "" {
		missing = append(missing, "BITBUCKET_APP_PASSWORD")
	}
	if strings.TrimSpace(c.BitbucketWorkspace) == "" {
		missing = append(missing, "BITBUCKET_WORKSPACE")
	}
	if strings.TrimSpace(c.GitHubToken) == "" {
		missing = append(missing, "GITHUB_TOKEN")
	}
	if strings.TrimSpace(c.GitHubOwner) == "" {
		missing = append(missing, "GITHUB_OWNER")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}
	return nil
}

func ConfigureInteractive(in io.Reader, out io.Writer, path string, kr Keyring) (Config, error) {
	reader := bufio.NewReader(in)
	return configureInteractiveFromReader(reader, in, out, path, kr)
}

func ConfigureInteractiveIfAllowed(in io.Reader, out io.Writer, path string, kr Keyring) (Config, bool, error) {
	if _, err := os.Stat(path); err == nil {
		reader := bufio.NewReader(in)
		fmt.Fprintf(out, "%s already exists. Overwrite? [y/N]: ", filepath.Base(path))
		answer, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return Config{}, false, err
		}
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(out, "Configuration unchanged.")
			return Config{}, false, nil
		}
		return configureWithReader(reader, in, out, path, kr)
	} else if !os.IsNotExist(err) {
		return Config{}, false, fmt.Errorf("failed to check %s: %w", path, err)
	}
	cfg, err := ConfigureInteractive(in, out, path, kr)
	return cfg, err == nil, err
}

func configureWithReader(reader *bufio.Reader, rawIn io.Reader, out io.Writer, path string, kr Keyring) (Config, bool, error) {
	cfg, err := configureInteractiveFromReader(reader, rawIn, out, path, kr)
	return cfg, err == nil, err
}

func configureInteractiveFromReader(reader *bufio.Reader, rawIn io.Reader, out io.Writer, path string, kr Keyring) (Config, error) {
	ask := func(label string) (string, error) {
		fmt.Fprintf(out, "%s: ", label)
		value, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		return strings.TrimSpace(value), nil
	}

	PrintCredentialGuidance(out)
	cfg := Config{}
	var err error
	if cfg.BitbucketUsername, err = ask("Bitbucket username (email)"); err != nil {
		return Config{}, err
	}
	if cfg.BitbucketAppPassword, err = askSecret(reader, rawIn, out, "Bitbucket app password (hidden)"); err != nil {
		return Config{}, err
	}
	if cfg.BitbucketWorkspace, err = ask("Bitbucket workspace"); err != nil {
		return Config{}, err
	}
	if cfg.GitHubToken, err = askSecret(reader, rawIn, out, "GitHub token (hidden)"); err != nil {
		return Config{}, err
	}
	if cfg.GitHubOwner, err = ask("GitHub owner/org"); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	if err := Write(path, cfg, kr); err != nil {
		return Config{}, err
	}
	fmt.Fprintf(out, "Wrote encrypted config %s\n", path)
	return cfg, nil
}

func askSecret(reader *bufio.Reader, rawIn io.Reader, out io.Writer, label string) (string, error) {
	fmt.Fprintf(out, "%s: ", label)
	if file, ok := rawIn.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		value, err := term.ReadPassword(int(file.Fd()))
		fmt.Fprintln(out)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(value)), nil
	}
	value, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func PrintCredentialGuidance(out io.Writer) {
	fmt.Fprintln(out, "Required token permissions:")
	fmt.Fprintln(out, "- Bitbucket app password permissions:")
	fmt.Fprintln(out, "  - Account: Read")
	fmt.Fprintln(out, "  - Workspace membership: Read")
	fmt.Fprintln(out, "  - Projects: Read")
	fmt.Fprintln(out, "  - Repositories: Read")
	fmt.Fprintln(out, "- GitHub token permissions:")
	fmt.Fprintln(out, "  - Metadata: Read-only")
	fmt.Fprintln(out, "  - Administration: Read and write")
	fmt.Fprintln(out, "  - Contents: Read and write")
	fmt.Fprintln(out, "")
}

func HasConfig(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func encodeEncrypted(cfg Config, kr Keyring) ([]byte, error) {
	key, err := getOrCreateKey(kr)
	if err != nil {
		return nil, err
	}
	plain, err := json.Marshal(fileConfig{
		BitbucketUsername:    cfg.BitbucketUsername,
		BitbucketAppPassword: cfg.BitbucketAppPassword,
		BitbucketWorkspace:   cfg.BitbucketWorkspace,
		GitHubToken:          cfg.GitHubToken,
		GitHubOwner:          cfg.GitHubOwner,
	})
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ciphertext := aead.Seal(nil, nonce, plain, nil)
	return marshalEncryptedFile(encryptedFile{
		Version:    1,
		Encryption: encryptionMethod,
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Data:       base64.StdEncoding.EncodeToString(ciphertext),
	}), nil
}

func readEncrypted(r io.Reader, kr Keyring) (Config, error) {
	envelope, err := parseEncryptedFile(r)
	if err != nil {
		return Config{}, err
	}
	if envelope.Version != 1 {
		return Config{}, fmt.Errorf("unsupported config version %d", envelope.Version)
	}
	if envelope.Encryption != encryptionMethod {
		return Config{}, fmt.Errorf("unsupported config encryption %q", envelope.Encryption)
	}
	key, err := getKey(kr)
	if err != nil {
		return Config{}, err
	}
	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return Config{}, fmt.Errorf("invalid config nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Data)
	if err != nil {
		return Config{}, fmt.Errorf("invalid config data: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return Config{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return Config{}, err
	}
	plain, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return Config{}, fmt.Errorf("failed to decrypt config: %w", err)
	}
	fileCfg := fileConfig{}
	if err := json.Unmarshal(plain, &fileCfg); err != nil {
		return Config{}, fmt.Errorf("failed to decode config payload: %w", err)
	}
	return Config{
		BitbucketUsername:    fileCfg.BitbucketUsername,
		BitbucketAppPassword: fileCfg.BitbucketAppPassword,
		BitbucketWorkspace:   fileCfg.BitbucketWorkspace,
		GitHubToken:          fileCfg.GitHubToken,
		GitHubOwner:          fileCfg.GitHubOwner,
	}, nil
}

func getKey(kr Keyring) ([]byte, error) {
	encoded, err := kr.Get(keyringService, keyringAccount)
	if err != nil {
		return nil, err
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("stored config key is invalid: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("stored config key has invalid length %d", len(key))
	}
	return key, nil
}

func getOrCreateKey(kr Keyring) ([]byte, error) {
	key, err := getKey(kr)
	if err == nil {
		return key, nil
	}
	if !errors.Is(err, ErrKeyNotFound) {
		return nil, err
	}
	key = make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := kr.Set(keyringService, keyringAccount, base64.StdEncoding.EncodeToString(key)); err != nil {
		return nil, err
	}
	return key, nil
}

func marshalEncryptedFile(file encryptedFile) []byte {
	return []byte(fmt.Sprintf("version: %d\nencryption: %s\nnonce: %s\ndata: %s\n", file.Version, file.Encryption, file.Nonce, file.Data))
}

func parseEncryptedFile(r io.Reader) (encryptedFile, error) {
	scanner := bufio.NewScanner(r)
	values := map[string]string{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return encryptedFile{}, fmt.Errorf("invalid config line %q", line)
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if err := scanner.Err(); err != nil {
		return encryptedFile{}, err
	}
	version := 0
	if values["version"] == "1" {
		version = 1
	}
	return encryptedFile{
		Version:    version,
		Encryption: strings.Trim(values["encryption"], `"'`),
		Nonce:      strings.Trim(values["nonce"], `"'`),
		Data:       strings.Trim(values["data"], `"'`),
	}, nil
}

func applyEnvOverrides(cfg *Config) {
	values := map[string]*string{
		"BITBUCKET_USERNAME":     &cfg.BitbucketUsername,
		"BITBUCKET_APP_PASSWORD": &cfg.BitbucketAppPassword,
		"BITBUCKET_WORKSPACE":    &cfg.BitbucketWorkspace,
		"GITHUB_TOKEN":           &cfg.GitHubToken,
		"GITHUB_OWNER":           &cfg.GitHubOwner,
	}
	for _, key := range envKeys() {
		if value, ok := os.LookupEnv(key); ok {
			*values[key] = value
		}
	}
}

func envKeys() []string {
	keys := []string{
		"BITBUCKET_USERNAME",
		"BITBUCKET_APP_PASSWORD",
		"BITBUCKET_WORKSPACE",
		"GITHUB_TOKEN",
		"GITHUB_OWNER",
	}
	sort.Strings(keys)
	return keys
}
