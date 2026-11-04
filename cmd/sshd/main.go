// Commande sshd expose le client terminal via SSH grâce à Wish.
// La connexion SSH ouvre l'interface. L'authentification du compte
// (email et mot de passe) se fait ensuite dans cette interface, via l'API HTTP.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"

	"github.com/madoxSio/go-cli-market/internal/config"
	"github.com/madoxSio/go-cli-market/internal/tui"
)

func main() {
	cfg := config.Load()
	if err := os.MkdirAll(filepath.Dir(cfg.SSHHostKey), 0o700); err != nil {
		slog.Error("dossier de la clé SSH", "err", err)
		os.Exit(1)
	}

	// Le dernier middleware est exécuté le premier : on journalise, on exige un
	// terminal actif, puis Bubble Tea prend la session.
	server, err := wish.NewServer(
		wish.WithAddress(cfg.SSHAddr),
		wish.WithHostKeyPath(cfg.SSHHostKey),
		wish.WithBanner("go-cli-market"),
		wish.WithMiddleware(
			bubbletea.Middleware(func(ssh.Session) (tea.Model, []tea.ProgramOption) {
				return tui.NewModel(cfg.APIURL), []tea.ProgramOption{
					tea.WithAltScreen(),
					tea.WithReportFocus(),
				}
			}),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		slog.Error("serveur SSH", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shut); err != nil {
			slog.Error("arrêt SSH", "err", err)
		}
	}()

	slog.Info("client SSH démarré", "addr", cfg.SSHAddr, "api", cfg.APIURL)
	slog.Info("connexion", "exemple", fmt.Sprintf("ssh -p %s localhost", portOf(cfg.SSHAddr)))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		slog.Error("arrêt du serveur SSH", "err", err)
		os.Exit(1)
	}
}

func portOf(addr string) string {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[i+1:]
		}
	}
	return addr
}
