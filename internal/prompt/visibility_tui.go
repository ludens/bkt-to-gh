package prompt

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/ludens/bkt-to-gh/internal/policy"
)

func ChooseVisibilityPolicyAuto(in *os.File, out *os.File) (policy.VisibilityPolicy, error) {
	if in == nil || out == nil || !term.IsTerminal(int(in.Fd())) || !term.IsTerminal(int(out.Fd())) {
		return ChooseVisibilityPolicy(in, out)
	}
	m := newVisibilityPolicyModel()
	program := tea.NewProgram(m, tea.WithInput(in), tea.WithOutput(out))
	result, err := program.Run()
	if err != nil {
		return "", fmt.Errorf("visibility selector failed: %w", err)
	}
	final, ok := result.(visibilityPolicyModel)
	if !ok {
		return "", fmt.Errorf("visibility selector returned unexpected model")
	}
	if final.canceled {
		return "", fmt.Errorf("visibility selection canceled")
	}
	return final.selectedPolicy(), nil
}

type visibilityPolicyModel struct {
	cursor   int
	done     bool
	canceled bool
}

type visibilityOption struct {
	policy      policy.VisibilityPolicy
	label       string
	description string
}

var visibilityOptions = []visibilityOption{
	{policy: policy.AllPrivate, label: "all-private", description: "create every GitHub repository as private"},
	{policy: policy.AllPublic, label: "all-public", description: "create every GitHub repository as public"},
	{policy: policy.FollowSource, label: "follow-source", description: "keep each Bitbucket repository visibility"},
}

func newVisibilityPolicyModel() visibilityPolicyModel {
	return visibilityPolicyModel{}
}

func (m visibilityPolicyModel) Init() tea.Cmd {
	return nil
}

func (m visibilityPolicyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.canceled = true
		return m, tea.Quit
	case tea.KeyEnter:
		m.done = true
		return m, tea.Quit
	case tea.KeyUp:
		m.move(-1)
	case tea.KeyDown:
		m.move(1)
	case tea.KeyRunes:
		switch strings.ToLower(key.String()) {
		case "k":
			m.move(-1)
		case "j":
			m.move(1)
		case "q":
			m.canceled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m visibilityPolicyModel) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, "Choose GitHub visibility policy")
	fmt.Fprintln(&b, "")
	for i, option := range visibilityOptions {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		fmt.Fprintf(&b, "%s %-14s %s\n", cursor, option.label, option.description)
	}
	fmt.Fprintln(&b, "")
	fmt.Fprintln(&b, "↑/↓ move  enter confirm  q cancel")
	return b.String()
}

func (m *visibilityPolicyModel) move(delta int) {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = len(visibilityOptions) - 1
	}
	if m.cursor >= len(visibilityOptions) {
		m.cursor = 0
	}
}

func (m visibilityPolicyModel) selectedPolicy() policy.VisibilityPolicy {
	return visibilityOptions[m.cursor].policy
}

var _ tea.Model = visibilityPolicyModel{}
