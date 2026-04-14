package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/term"
)

type Config struct {
	BitbucketUsername    string
	BitbucketAppPassword string
	BitbucketWorkspace   string
	GitHubToken          string
	GitHubOwner          string
}

func Load(dir string) (Config, error) {
	values := map[string]string{}
	envPath := filepath.Join(dir, ".env")
	if file, err := os.Open(envPath); err == nil {
		defer file.Close()
		parsed, err := parseDotEnv(file)
		if err != nil {
			return Config{}, fmt.Errorf("failed to parse %s: %w", envPath, err)
		}
		for key, value := range parsed {
			values[key] = value
		}
	} else if !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("failed to read %s: %w", envPath, err)
	}

	for _, key := range envKeys() {
		if value, ok := os.LookupEnv(key); ok {
			values[key] = value
		}
	}

	return Config{
		BitbucketUsername:    values["BITBUCKET_USERNAME"],
		BitbucketAppPassword: values["BITBUCKET_APP_PASSWORD"],
		BitbucketWorkspace:   values["BITBUCKET_WORKSPACE"],
		GitHubToken:          values["GITHUB_TOKEN"],
		GitHubOwner:          values["GITHUB_OWNER"],
	}, nil
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

func WriteDotEnv(path string, cfg Config) error {
	content := strings.Join([]string{
		"BITBUCKET_USERNAME=" + cfg.BitbucketUsername,
		"BITBUCKET_APP_PASSWORD=" + cfg.BitbucketAppPassword,
		"BITBUCKET_WORKSPACE=" + cfg.BitbucketWorkspace,
		"GITHUB_TOKEN=" + cfg.GitHubToken,
		"GITHUB_OWNER=" + cfg.GitHubOwner,
		"",
	}, "\n")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

func ConfigureInteractive(in io.Reader, out io.Writer, path string) (Config, error) {
	reader := bufio.NewReader(in)
	return configureInteractiveFromReader(reader, in, out, path)
}

func ConfigureInteractiveIfAllowed(in io.Reader, out io.Writer, path string) (Config, bool, error) {
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
		return configureWithReader(reader, in, out, path)
	} else if !os.IsNotExist(err) {
		return Config{}, false, fmt.Errorf("failed to check %s: %w", path, err)
	}
	cfg, err := ConfigureInteractive(in, out, path)
	return cfg, err == nil, err
}

func configureWithReader(reader *bufio.Reader, rawIn io.Reader, out io.Writer, path string) (Config, bool, error) {
	cfg, err := configureInteractiveFromReader(reader, rawIn, out, path)
	return cfg, err == nil, err
}

func configureInteractiveFromReader(reader *bufio.Reader, rawIn io.Reader, out io.Writer, path string) (Config, error) {
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
	if err := WriteDotEnv(path, cfg); err != nil {
		return Config{}, err
	}
	fmt.Fprintf(out, "Wrote %s\n", path)
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

func HasDotEnv(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".env"))
	return err == nil
}

func parseDotEnv(r io.Reader) (map[string]string, error) {
	scanner := bufio.NewScanner(r)
	values := map[string]string{}
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected KEY=value", lineNo)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("line %d: empty key", lineNo)
		}
		values[key] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
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
