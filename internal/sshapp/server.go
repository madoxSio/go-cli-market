// Package sshapp démarre un serveur Wish qui affiche une interface Bubble Tea.
package sshapp

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
)

// Serve écoute en SSH et lance newModel pour chaque session.
// Le dernier middleware est exécuté le premier.
func Serve(addr, hostKey, banner, label string, newModel func() tea.Model) error {
	if err := os.MkdirAll(filepath.Dir(hostKey), 0o700); err != nil {
		return err
	}
	server, err := wish.NewServer(
		wish.WithAddress(addr),
		wish.WithHostKeyPath(hostKey),
		wish.WithBanner(banner),
		wish.WithMiddleware(
			bubbletea.Middleware(func(ssh.Session) (tea.Model, []tea.ProgramOption) {
				return newModel(), []tea.ProgramOption{tea.WithAltScreen(), tea.WithReportFocus()}
			}),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shut); err != nil {
			slog.Error("arrêt SSH", "interface", label, "err", err)
		}
	}()

	slog.Info("SSH démarré", "interface", label, "addr", addr)
	slog.Info("connexion", "exemple", fmt.Sprintf("ssh -p %s localhost", portOf(addr)))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		return err
	}
	return nil
}

func portOf(addr string) string {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[i+1:]
		}
	}
	return addr
}
