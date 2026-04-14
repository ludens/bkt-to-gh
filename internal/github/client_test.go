package github

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateRepositoryCreatesPrivateRepo(t *testing.T) {
	var body string
	paths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/user" {
			w.Write([]byte(`{"login":"token-user"}`))
			return
		}
		if r.URL.Path != "/orgs/acme/repos" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("Authorization = %q", got)
		}
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		body = string(buf)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"clone_url":"https://github.com/acme/repo-one.git"}`))
	}))
	defer server.Close()

	client := NewClient("token", "acme")
	client.BaseURL = server.URL

	result, err := client.CreateRepository(context.Background(), CreateRepositoryRequest{
		Name:        "repo-one",
		Description: "desc",
		Private:     true,
	})
	if err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}
	if result.Skipped {
		t.Fatal("Skipped = true, want false")
	}
	if result.CloneURL != "https://github.com/acme/repo-one.git" {
		t.Fatalf("CloneURL = %q", result.CloneURL)
	}
	if strings.Join(paths, ",") != "/user,/orgs/acme/repos" {
		t.Fatalf("paths = %v", paths)
	}
	for _, want := range []string{`"name":"repo-one"`, `"description":"desc"`, `"private":true`} {
		if !strings.Contains(body, want) {
			t.Fatalf("request body missing %s in %s", want, body)
		}
	}
}

func TestCreateRepositoryForAuthenticatedUserUsesUserRepos(t *testing.T) {
	paths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/user":
			w.Write([]byte(`{"login":"ludens"}`))
		case "/user/repos":
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"clone_url":"https://github.com/ludens/repo-one.git"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient("token", "ludens")
	client.BaseURL = server.URL

	result, err := client.CreateRepository(context.Background(), CreateRepositoryRequest{Name: "repo-one"})
	if err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}
	if result.CloneURL != "https://github.com/ludens/repo-one.git" {
		t.Fatalf("CloneURL = %q", result.CloneURL)
	}
	if strings.Join(paths, ",") != "/user,/user/repos" {
		t.Fatalf("paths = %v", paths)
	}
}

func TestCreateRepositorySkipsExisting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user" {
			w.Write([]byte(`{"login":"token-user"}`))
			return
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"message":"Repository creation failed.","errors":[{"message":"name already exists on this account"}]}`))
	}))
	defer server.Close()

	client := NewClient("token", "acme")
	client.BaseURL = server.URL

	result, err := client.CreateRepository(context.Background(), CreateRepositoryRequest{Name: "repo-one"})
	if err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}
	if !result.Skipped {
		t.Fatal("Skipped = false, want true")
	}
}

func TestRepositoryExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/repo-one" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("token", "acme")
	client.BaseURL = server.URL

	exists, err := client.RepositoryExists(context.Background(), "repo-one")
	if err != nil {
		t.Fatalf("RepositoryExists() error = %v", err)
	}
	if !exists {
		t.Fatal("exists = false, want true")
	}
}

func TestRepositoryExistsReturnsFalseForNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient("token", "acme")
	client.BaseURL = server.URL

	exists, err := client.RepositoryExists(context.Background(), "missing")
	if err != nil {
		t.Fatalf("RepositoryExists() error = %v", err)
	}
	if exists {
		t.Fatal("exists = true, want false")
	}
}

func TestCheckCreateAccessChecksUserAndOrg(t *testing.T) {
	paths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/user" {
			w.Write([]byte(`{"login":"token-user"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient("token", "acme")
	client.BaseURL = server.URL

	if err := client.CheckCreateAccess(context.Background()); err != nil {
		t.Fatalf("CheckCreateAccess() error = %v", err)
	}
	if strings.Join(paths, ",") != "/user,/orgs/acme" {
		t.Fatalf("paths = %v", paths)
	}
}

func TestCheckCreateAccessAllowsAuthenticatedUserOwner(t *testing.T) {
	paths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/user" {
			w.Write([]byte(`{"login":"ludens"}`))
			return
		}
		t.Fatalf("unexpected path %s", r.URL.Path)
	}))
	defer server.Close()

	client := NewClient("token", "ludens")
	client.BaseURL = server.URL

	if err := client.CheckCreateAccess(context.Background()); err != nil {
		t.Fatalf("CheckCreateAccess() error = %v", err)
	}
	if strings.Join(paths, ",") != "/user" {
		t.Fatalf("paths = %v", paths)
	}
}
