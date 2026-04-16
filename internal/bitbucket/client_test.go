package bitbucket

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListRepositoriesPaginates(t *testing.T) {
	var seenAuth bool
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if ok && user == "user" && pass == "pass" {
			seenAuth = true
		}
		if r.URL.Path != "/2.0/repositories/team" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		switch r.URL.Query().Get("page") {
		case "":
			w.Write([]byte(`{
				"values": [
					{
						"name": "Repo One",
						"slug": "repo-one",
						"is_private": true,
						"description": "first",
						"project": {"name": "Project A"},
						"links": {"clone": [{"name": "https", "href": "https://bitbucket.org/team/repo-one.git"}]}
					}
				],
				"next": "` + server.URL + `/2.0/repositories/team?page=2"
			}`))
		case "2":
			w.Write([]byte(`{
				"values": [
					{
						"name": "Repo Two",
						"slug": "repo-two",
						"is_private": false,
						"description": "",
						"links": {"clone": [{"name": "ssh", "href": "git@bitbucket.org:team/repo-two.git"}]}
					}
				]
			}`))
		default:
			t.Fatalf("unexpected page %q", r.URL.Query().Get("page"))
		}
	}))
	defer server.Close()

	client := NewClient("user", "pass")
	client.BaseURL = server.URL

	repos, err := client.ListRepositories(context.Background(), "team")
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}
	if !seenAuth {
		t.Fatal("server did not see expected basic auth")
	}
	if len(repos) != 2 {
		t.Fatalf("len(repos) = %d, want 2", len(repos))
	}
	if repos[0].Slug != "repo-one" || !repos[0].Private || repos[0].ProjectName != "Project A" {
		t.Fatalf("repo[0] = %+v", repos[0])
	}
	if repos[1].CloneURL != "git@bitbucket.org:team/repo-two.git" {
		t.Fatalf("repo[1].CloneURL = %q", repos[1].CloneURL)
	}
}

func TestListRepositoriesClosesResponseBodyBeforeNextPage(t *testing.T) {
	transport := &closeCountingTransport{}
	client := NewClient("user", "pass")
	client.BaseURL = "https://bitbucket.example.test"
	client.HTTPClient = &http.Client{Transport: transport}

	_, err := client.ListRepositories(context.Background(), "team")
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}
	if len(transport.bodies) != 2 {
		t.Fatalf("body count = %d, want 2", len(transport.bodies))
	}
	for i, body := range transport.bodies {
		if !body.closed {
			t.Fatalf("body %d was not closed", i)
		}
	}
}

type closeCountingTransport struct {
	bodies []*closeCountingBody
}

func (t *closeCountingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(t.bodies) == 1 && !t.bodies[0].closed {
		return nil, fmt.Errorf("first response body was not closed before next page request")
	}
	bodyText := `{"values":[]}`
	if len(t.bodies) == 0 {
		bodyText = `{"values":[],"next":"https://bitbucket.example.test/2.0/repositories/team?page=2"}`
	}
	body := &closeCountingBody{Reader: strings.NewReader(bodyText)}
	t.bodies = append(t.bodies, body)
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       body,
		Request:    req,
	}, nil
}

type closeCountingBody struct {
	*strings.Reader
	closed bool
}

func (b *closeCountingBody) Close() error {
	b.closed = true
	return nil
}

var _ io.ReadCloser = (*closeCountingBody)(nil)
