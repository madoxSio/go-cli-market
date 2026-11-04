// Commande admin-sshd expose l'administration via SSH, à part du client.
package main

import (
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madoxSio/go-cli-market/internal/admintui"
	"github.com/madoxSio/go-cli-market/internal/config"
	"github.com/madoxSio/go-cli-market/internal/sshapp"
)

func main() {
	cfg := config.Load()
	err := sshapp.Serve(cfg.AdminSSHAddr, cfg.AdminSSHHostKey, "go-cli-market admin", "admin", func() tea.Model {
		return admintui.NewModel(cfg.APIURL)
	})
	if err != nil {
		slog.Error("serveur SSH admin", "err", err)
		os.Exit(1)
	}
}
