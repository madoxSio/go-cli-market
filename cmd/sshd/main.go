// Commande sshd expose le client terminal via SSH grâce à Wish.
package main

import (
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/madoxSio/go-cli-market/internal/config"
	"github.com/madoxSio/go-cli-market/internal/sshapp"
	"github.com/madoxSio/go-cli-market/internal/tui"
)

func main() {
	cfg := config.Load()
	err := sshapp.Serve(cfg.SSHAddr, cfg.SSHHostKey, "go-cli-market client", "client", func() tea.Model {
		return tui.NewModel(cfg.APIURL)
	})
	if err != nil {
		slog.Error("serveur SSH client", "err", err)
		os.Exit(1)
	}
}
