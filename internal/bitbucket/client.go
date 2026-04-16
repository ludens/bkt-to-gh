package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ludens/bkt-to-gh/internal/model"
)

const defaultBaseURL = "https://api.bitbucket.org"

type Client struct {
	BaseURL     string
	Username    string
	AppPassword string
	HTTPClient  *http.Client
}

func NewClient(username, appPassword string) *Client {
	return &Client{
		BaseURL:     defaultBaseURL,
		Username:    username,
		AppPassword: appPassword,
		HTTPClient:  http.DefaultClient,
	}
}

func (c *Client) ListRepositories(ctx context.Context, workspace string) ([]model.Repository, error) {
	if workspace == "" {
		return nil, fmt.Errorf("bitbucket workspace is required")
	}
	nextURL := fmt.Sprintf("%s/2.0/repositories/%s?pagelen=100", c.BaseURL, url.PathEscape(workspace))
	repos := []model.Repository{}

	for nextURL != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, nextURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to build Bitbucket request: %w", err)
		}
		req.SetBasicAuth(c.Username, c.AppPassword)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient().Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to call Bitbucket API: %w", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			resp.Body.Close()
			return nil, fmt.Errorf("Bitbucket API returned %s while listing repositories", resp.Status)
		}

		var page repositoriesPage
		decodeErr := json.NewDecoder(resp.Body).Decode(&page)
		closeErr := resp.Body.Close()
		if decodeErr != nil {
			return nil, fmt.Errorf("failed to decode Bitbucket repositories response: %w", decodeErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("failed to close Bitbucket repositories response: %w", closeErr)
		}
		for _, item := range page.Values {
			repos = append(repos, item.toModel())
		}
		nextURL = page.Next
	}

	return repos, nil
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

type repositoriesPage struct {
	Values []repository `json:"values"`
	Next   string       `json:"next"`
}

type repository struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	IsPrivate   bool   `json:"is_private"`
	Description string `json:"description"`
	Project     struct {
		Name string `json:"name"`
	} `json:"project"`
	Links struct {
		Clone []struct {
			Name string `json:"name"`
			Href string `json:"href"`
		} `json:"clone"`
	} `json:"links"`
}

func (r repository) toModel() model.Repository {
	return model.Repository{
		Name:        r.Name,
		Slug:        r.Slug,
		Private:     r.IsPrivate,
		Description: r.Description,
		ProjectName: r.Project.Name,
		CloneURL:    pickCloneURL(r),
	}
}

func pickCloneURL(r repository) string {
	if len(r.Links.Clone) == 0 {
		return ""
	}
	for _, clone := range r.Links.Clone {
		if clone.Name == "https" {
			return clone.Href
		}
	}
	return r.Links.Clone[0].Href
}
