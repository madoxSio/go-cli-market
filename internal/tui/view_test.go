package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/madoxSio/go-cli-market/internal/apiclient"
)

func TestViewsFitAn80ColumnTerminal(t *testing.T) {
	m := NewModel("http://localhost:8080")
	m.width = 80
	m.height = 24
	assertFits(t, "accueil", m.View(), 80, 24)

	m.authed = true
	m.session = apiclient.User{Email: "ada@example.com", Role: "client"}
	m.useSessionMenu()
	assertFits(t, "session", m.View(), 80, 24)

	m.screen = screenNotice
	m.noticeLead = "Conservez ce code pour confirmer le compte."
	m.pendingCode = "ABCD2345"
	assertFits(t, "code", m.View(), 80, 24)

	m.screen = screenHelp
	m.renderHelp()
	assertFits(t, "aide", m.View(), 80, 24)

	m.screen = screenLoading
	m.loadingText = "Connexion..."
	assertFits(t, "chargement", m.View(), 80, 24)
}

func TestArrowKeysMoveAcrossTheCardGrid(t *testing.T) {
	m := NewModel("http://localhost:8080")
	m.width = 80
	m.height = 24
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	moved := next.(model)
	if moved.cursor != 1 {
		t.Fatalf("curseur %d, attendu 1", moved.cursor)
	}
	next, _ = moved.Update(tea.KeyMsg{Type: tea.KeyDown})
	moved = next.(model)
	if moved.cursor != 3 {
		t.Fatalf("curseur %d, attendu 3", moved.cursor)
	}
}

func assertFits(t *testing.T, name, view string, width, height int) {
	t.Helper()
	if got := lipgloss.Width(view); got > width {
		t.Errorf("%s : largeur %d, maximum %d", name, got, width)
	}
	if got := lipgloss.Height(view); got > height {
		t.Errorf("%s : hauteur %d, maximum %d", name, got, height)
	}
}
