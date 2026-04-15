package prompt

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ludens/bkt-to-gh/internal/model"
)

func TestRepositorySelectorModelSelectsWithSpaceAndEnter(t *testing.T) {
	repos := []model.Repository{
		{Name: "API", Slug: "api"},
		{Name: "Web", Slug: "web"},
	}
	m := newRepositorySelectorModel(repos)

	modelAfter, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = modelAfter.(repositorySelectorModel)
	modelAfter, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = modelAfter.(repositorySelectorModel)
	modelAfter, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = modelAfter.(repositorySelectorModel)
	modelAfter, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = modelAfter.(repositorySelectorModel)

	selected := m.selectedRepositories()
	if len(selected) != 2 {
		t.Fatalf("len(selected) = %d, want 2", len(selected))
	}
	if selected[0].Slug != "api" || selected[1].Slug != "web" {
		t.Fatalf("selected = %+v", selected)
	}
	if !m.done {
		t.Fatal("done = false, want true")
	}
}

func TestRepositorySelectorModelFiltersAndSelectsVisibleItem(t *testing.T) {
	repos := []model.Repository{
		{Name: "API", Slug: "api"},
		{Name: "Worker", Slug: "worker"},
	}
	m := newRepositorySelectorModel(repos)

	for _, msg := range []tea.Msg{
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("work")},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeySpace},
		tea.KeyMsg{Type: tea.KeyEnter},
	} {
		modelAfter, _ := m.Update(msg)
		m = modelAfter.(repositorySelectorModel)
	}

	selected := m.selectedRepositories()
	if len(selected) != 1 || selected[0].Slug != "worker" {
		t.Fatalf("selected = %+v, want worker", selected)
	}
}

func TestRepositorySelectorModelAllAndNone(t *testing.T) {
	repos := []model.Repository{
		{Name: "API", Slug: "api"},
		{Name: "Web", Slug: "web"},
	}
	m := newRepositorySelectorModel(repos)

	modelAfter, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = modelAfter.(repositorySelectorModel)
	if len(m.selectedRepositories()) != 2 {
		t.Fatalf("selected count after all = %d, want 2", len(m.selectedRepositories()))
	}

	modelAfter, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = modelAfter.(repositorySelectorModel)
	if len(m.selectedRepositories()) != 0 {
		t.Fatalf("selected count after none = %d, want 0", len(m.selectedRepositories()))
	}
}
