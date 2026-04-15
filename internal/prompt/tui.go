package prompt

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/ludens/bkt-to-gh/internal/model"
)

func SelectRepositoriesAuto(in *os.File, out *os.File, repos []model.Repository) ([]model.Repository, error) {
	if in == nil || out == nil || !term.IsTerminal(int(in.Fd())) || !term.IsTerminal(int(out.Fd())) {
		return SelectRepositories(in, out, repos)
	}
	m := newRepositorySelectorModel(repos)
	program := tea.NewProgram(m, tea.WithInput(in), tea.WithOutput(out))
	result, err := program.Run()
	if err != nil {
		return nil, fmt.Errorf("repository selector failed: %w", err)
	}
	final, ok := result.(repositorySelectorModel)
	if !ok {
		return nil, fmt.Errorf("repository selector returned unexpected model")
	}
	if final.canceled {
		return nil, fmt.Errorf("repository selection canceled")
	}
	return final.selectedRepositories(), nil
}

type repositorySelectorModel struct {
	repos       []model.Repository
	cursor      int
	selected    map[int]bool
	filter      string
	filterMode  bool
	filterInput string
	done        bool
	canceled    bool
}

func newRepositorySelectorModel(repos []model.Repository) repositorySelectorModel {
	return repositorySelectorModel{
		repos:    repos,
		selected: map[int]bool{},
	}
}

func (m repositorySelectorModel) Init() tea.Cmd {
	return nil
}

func (m repositorySelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if m.filterMode {
		return m.updateFilter(key)
	}

	switch key.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.canceled = true
		return m, tea.Quit
	case tea.KeyEnter:
		m.done = true
		return m, tea.Quit
	case tea.KeyUp:
		m.moveCursor(-1)
	case tea.KeyDown:
		m.moveCursor(1)
	case tea.KeySpace:
		m.toggleCurrent()
	case tea.KeyRunes:
		switch key.String() {
		case "k":
			m.moveCursor(-1)
		case "j":
			m.moveCursor(1)
		case "a":
			m.selectVisible(true)
		case "n":
			m.selectVisible(false)
		case "/":
			m.filterMode = true
			m.filterInput = m.filter
		case "q":
			m.canceled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m repositorySelectorModel) updateFilter(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyCtrlC:
		m.canceled = true
		return m, tea.Quit
	case tea.KeyEsc:
		m.filterMode = false
		m.filterInput = ""
	case tea.KeyEnter:
		m.filter = strings.TrimSpace(m.filterInput)
		m.filterMode = false
		m.cursor = 0
	case tea.KeyBackspace:
		if len(m.filterInput) > 0 {
			m.filterInput = m.filterInput[:len(m.filterInput)-1]
		}
	case tea.KeyRunes:
		m.filterInput += string(key.Runes)
	}
	return m, nil
}

func (m repositorySelectorModel) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, "Select repositories")
	fmt.Fprintln(&b, "")
	visible := m.visibleIndexes()
	if len(visible) == 0 {
		fmt.Fprintln(&b, "  No repositories match current filter.")
	} else {
		for row, idx := range visible {
			repo := m.repos[idx]
			cursor := " "
			if row == m.cursor {
				cursor = ">"
			}
			mark := " "
			if m.selected[idx] {
				mark = "x"
			}
			fmt.Fprintf(&b, "%s [%s] %-28s %s\n", cursor, mark, repo.Slug, visibilityText(repo.Private))
		}
	}
	fmt.Fprintln(&b, "")
	if m.filterMode {
		fmt.Fprintf(&b, "Filter: %s\n", m.filterInput)
		fmt.Fprintln(&b, "enter apply  esc cancel")
	} else {
		if m.filter != "" {
			fmt.Fprintf(&b, "Filter: %q\n", m.filter)
		}
		fmt.Fprintf(&b, "Selected: %d\n", len(m.selectedRepositories()))
		fmt.Fprintln(&b, "↑/↓ move  space select  / filter  a all  n none  enter confirm  q cancel")
	}
	return b.String()
}

func (m *repositorySelectorModel) moveCursor(delta int) {
	visible := m.visibleIndexes()
	if len(visible) == 0 {
		m.cursor = 0
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = len(visible) - 1
	}
	if m.cursor >= len(visible) {
		m.cursor = 0
	}
}

func (m *repositorySelectorModel) toggleCurrent() {
	visible := m.visibleIndexes()
	if len(visible) == 0 {
		return
	}
	idx := visible[m.cursor]
	m.selected[idx] = !m.selected[idx]
}

func (m *repositorySelectorModel) selectVisible(selectAll bool) {
	for _, idx := range m.visibleIndexes() {
		if selectAll {
			m.selected[idx] = true
		} else {
			delete(m.selected, idx)
		}
	}
}

func (m repositorySelectorModel) selectedRepositories() []model.Repository {
	out := []model.Repository{}
	for i, repo := range m.repos {
		if m.selected[i] {
			out = append(out, repo)
		}
	}
	return out
}

func (m repositorySelectorModel) visibleIndexes() []int {
	return visibleIndexes(m.repos, m.filter)
}

func visibilityText(private bool) string {
	if private {
		return "private"
	}
	return "public"
}

var _ tea.Model = repositorySelectorModel{}
