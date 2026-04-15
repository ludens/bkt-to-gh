package prompt

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ludens/bkt-to-gh/internal/policy"
)

func TestVisibilityPolicyModelSelectsWithEnter(t *testing.T) {
	m := newVisibilityPolicyModel()

	modelAfter, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = modelAfter.(visibilityPolicyModel)
	modelAfter, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = modelAfter.(visibilityPolicyModel)
	modelAfter, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = modelAfter.(visibilityPolicyModel)

	if !m.done {
		t.Fatal("done = false, want true")
	}
	if m.selectedPolicy() != policy.FollowSource {
		t.Fatalf("selectedPolicy() = %q, want follow-source", m.selectedPolicy())
	}
}

func TestVisibilityPolicyModelCancelsWithQ(t *testing.T) {
	m := newVisibilityPolicyModel()

	modelAfter, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = modelAfter.(visibilityPolicyModel)

	if !m.canceled {
		t.Fatal("canceled = false, want true")
	}
}
