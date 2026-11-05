// Commande server lance l'API HTTP de la plateforme.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/madoxSio/go-cli-market/internal/auth"
	"github.com/madoxSio/go-cli-market/internal/config"
	"github.com/madoxSio/go-cli-market/internal/httpapi"
	"github.com/madoxSio/go-cli-market/internal/store"
)

func main() {
	cfg := config.Load()

	db, err := store.Open(cfg.DatabaseURL)
	if err != nil {
		slog.Error("connexion PostgreSQL", "err", err)
		slog.Info("démarrez la base avec : make db-up")
		os.Exit(1)
	}
	defer db.Close()

	if err := store.Migrate(db); err != nil {
		slog.Error("migration", "err", err)
		os.Exit(1)
	}

	svc, err := auth.NewService(store.NewRepository(db))
	if err != nil {
		slog.Error("service d'authentification", "err", err)
		os.Exit(1)
	}
	if err := svc.EnsureAdmin(context.Background(), cfg.AdminEmail, cfg.AdminPassword); err != nil {
		slog.Error("compte administrateur", "err", err)
		os.Exit(1)
	}
	slog.Info("compte administrateur prêt", "email", cfg.AdminEmail)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewRouter(svc, store.NewMarket(db)),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shut); err != nil {
			slog.Error("arrêt HTTP", "err", err)
		}
	}()

	slog.Info("PostgreSQL prêt")
	slog.Info("serveur HTTP démarré", "addr", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("arrêt du serveur", "err", err)
		os.Exit(1)
	}
}
