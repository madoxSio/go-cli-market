package admintui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/madoxSio/go-cli-market/internal/apiclient"
	"github.com/madoxSio/go-cli-market/internal/market"
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

	m.orders = []market.Order{{
		Ref:          "CMD-1F2S8B",
		UserEmail:    "ada@example.com",
		Status:       market.StatusCancelled,
		CancelReason: "rupture de stock",
		TotalTTC:     2400,
		Items:        []market.Item{{Name: "Clavier", Quantity: 2, UnitPriceTTC: 1200, LineTTC: 2400}},
	}}
	m.useOrderList()
	assertFits(t, "commandes", m.View(), 80, 24)

	m.selectedOrder = m.orders[0]
	m.useOrderMenu()
	assertFits(t, "commande", m.View(), 80, 24)

	m.screen = screenHelp
	m.renderHelp()
	assertFits(t, "aide", m.View(), 80, 24)
}

func TestNextStatuses(t *testing.T) {
	got := nextStatuses(market.StatusPaid)
	if len(got) != 2 || got[0] != market.StatusShipping || got[1] != market.StatusCancelled {
		t.Errorf("payée : %v", got)
	}
	if got := nextStatuses(market.StatusDelivered); len(got) != 0 {
		t.Errorf("livrée devrait être final : %v", got)
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
