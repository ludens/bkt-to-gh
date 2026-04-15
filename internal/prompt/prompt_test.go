package prompt

import (
	"strings"
	"testing"

	"github.com/ludens/bkt-to-gh/internal/model"
)

func TestSelectRepositoriesSupportsFilterAllNoneAndDone(t *testing.T) {
	repos := []model.Repository{
		{Name: "API", Slug: "api"},
		{Name: "Web", Slug: "web"},
	}
	input := strings.NewReader("filter api\nall\nnone\n1\ndone\n")
	output := new(strings.Builder)

	selected, err := SelectRepositories(input, output, repos)
	if err != nil {
		t.Fatalf("SelectRepositories() error = %v", err)
	}
	if len(selected) != 1 || selected[0].Slug != "api" {
		t.Fatalf("selected = %+v, want only api", selected)
	}
}

func TestSelectRepositoriesSupportsCommaSeparatedNumbers(t *testing.T) {
	repos := []model.Repository{
		{Name: "API", Slug: "api"},
		{Name: "Web", Slug: "web"},
		{Name: "Worker", Slug: "worker"},
	}
	input := strings.NewReader("1,3\ndone\n")
	output := new(strings.Builder)

	selected, err := SelectRepositories(input, output, repos)
	if err != nil {
		t.Fatalf("SelectRepositories() error = %v", err)
	}
	if len(selected) != 2 {
		t.Fatalf("len(selected) = %d, want 2", len(selected))
	}
	if selected[0].Slug != "api" || selected[1].Slug != "worker" {
		t.Fatalf("selected = %+v, want api and worker", selected)
	}
	if !strings.Contains(output.String(), "1,3") {
		t.Fatalf("output should mention comma selection, got:\n%s", output.String())
	}
}

func TestChooseVisibilityPolicy(t *testing.T) {
	got, err := ChooseVisibilityPolicy(strings.NewReader("3\n"), new(strings.Builder))
	if err != nil {
		t.Fatalf("ChooseVisibilityPolicy() error = %v", err)
	}
	if string(got) != "follow-source" {
		t.Fatalf("policy = %q", got)
	}
}
