package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultBaseURL = "https://api.github.com"

type Client struct {
	BaseURL    string
	Token      string
	Owner      string
	HTTPClient *http.Client
}

type CreateRepositoryRequest struct {
	Name        string
	Description string
	Private     bool
}

type CreateRepositoryResult struct {
	CloneURL string
	Skipped  bool
	Reason   string
}

func NewClient(token, owner string) *Client {
	return &Client{
		BaseURL:    defaultBaseURL,
		Token:      token,
		Owner:      owner,
		HTTPClient: http.DefaultClient,
	}
}

func (c *Client) CreateRepository(ctx context.Context, input CreateRepositoryRequest) (CreateRepositoryResult, error) {
	if input.Name == "" {
		return CreateRepositoryResult{}, fmt.Errorf("GitHub repository name is required")
	}
	body, err := json.Marshal(map[string]any{
		"name":        input.Name,
		"description": input.Description,
		"private":     input.Private,
	})
	if err != nil {
		return CreateRepositoryResult{}, fmt.Errorf("failed to encode GitHub create repository request: %w", err)
	}

	createURL, err := c.createURL(ctx)
	if err != nil {
		return CreateRepositoryResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL, bytes.NewReader(body))
	if err != nil {
		return CreateRepositoryResult{}, fmt.Errorf("failed to build GitHub request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return CreateRepositoryResult{}, fmt.Errorf("failed to call GitHub API: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return CreateRepositoryResult{}, fmt.Errorf("failed to read GitHub response: %w", err)
	}

	if resp.StatusCode == http.StatusCreated {
		var created struct {
			CloneURL string `json:"clone_url"`
		}
		if err := json.Unmarshal(respBody, &created); err != nil {
			return CreateRepositoryResult{}, fmt.Errorf("failed to decode GitHub create repository response: %w", err)
		}
		return CreateRepositoryResult{CloneURL: created.CloneURL}, nil
	}
	if resp.StatusCode == http.StatusUnprocessableEntity && looksAlreadyExists(string(respBody)) {
		return CreateRepositoryResult{
			Skipped: true,
			Reason:  "GitHub repository already exists; overwrite is disabled",
		}, nil
	}

	return CreateRepositoryResult{}, fmt.Errorf("GitHub API returned %s while creating %s: %s", resp.Status, input.Name, trimBody(respBody))
}

func (c *Client) RepositoryExists(ctx context.Context, name string) (bool, error) {
	if c.Owner == "" {
		return false, fmt.Errorf("GITHUB_OWNER is required to check repository existence")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/repos/"+c.Owner+"/"+name, nil)
	if err != nil {
		return false, fmt.Errorf("failed to build GitHub repository check request: %w", err)
	}
	c.setAuthHeaders(req)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to call GitHub repository check API: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		respBody, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("GitHub API returned %s while checking %s: %s", resp.Status, name, trimBody(respBody))
	}
}

func (c *Client) CheckCreateAccess(ctx context.Context) error {
	login, err := c.currentUserLogin(ctx)
	if err != nil {
		return err
	}
	if c.Owner != "" && !strings.EqualFold(c.Owner, login) {
		if err := c.getOK(ctx, "/orgs/"+c.Owner, "checking GitHub owner access"); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) createURL(ctx context.Context) (string, error) {
	if c.Owner == "" {
		return c.BaseURL + "/user/repos", nil
	}
	login, err := c.currentUserLogin(ctx)
	if err != nil {
		return "", err
	}
	if strings.EqualFold(c.Owner, login) {
		return c.BaseURL + "/user/repos", nil
	}
	return c.BaseURL + "/orgs/" + c.Owner + "/repos", nil
}

func (c *Client) currentUserLogin(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/user", nil)
	if err != nil {
		return "", fmt.Errorf("failed to build GitHub request for checking GitHub token: %w", err)
	}
	c.setAuthHeaders(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call GitHub API for checking GitHub token: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read GitHub user response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("GitHub API returned %s while checking GitHub token: %s", resp.Status, trimBody(respBody))
	}
	var user struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal(respBody, &user); err != nil {
		return "", fmt.Errorf("failed to decode GitHub user response: %w", err)
	}
	if user.Login == "" {
		return "", fmt.Errorf("GitHub user response did not include login")
	}
	return user.Login, nil
}

func (c *Client) getOK(ctx context.Context, path, action string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to build GitHub request for %s: %w", action, err)
	}
	c.setAuthHeaders(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("failed to call GitHub API for %s: %w", action, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API returned %s while %s: %s", resp.Status, action, trimBody(respBody))
	}
	return nil
}

func (c *Client) setAuthHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func looksAlreadyExists(body string) bool {
	body = strings.ToLower(body)
	return strings.Contains(body, "already exists") || strings.Contains(body, "name already")
}

func trimBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	if len(text) > 500 {
		return text[:500] + "..."
	}
	return text
}
