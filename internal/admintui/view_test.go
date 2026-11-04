package admintui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/madoxSio/go-cli-market/internal/apiclient"
)

func TestViewsFitAn80ColumnTerminal(t *testing.T) {
	m := NewModel("http://localhost:8080")
	m.width = 80
	m.height = 24
	assertFits(t, "connexion", m.View(), 80, 24)

	m.authed = true
	m.session = apiclient.User{Email: "admin@market.local", Role: "admin"}
	m.screen = screenMenu
	m.useMainMenu()
	assertFits(t, "tableau", m.View(), 80, 24)

	m.users = []apiclient.User{{ID: 1, Email: "ada@example.com", Role: "client", Confirmed: true}}
	m.useUserList()
	assertFits(t, "comptes", m.View(), 80, 24)

	m.screen = screenHelp
	m.renderHelp()
	assertFits(t, "aide", m.View(), 80, 24)
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
