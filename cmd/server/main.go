// Commande server lance l'API HTTP de la plateforme.
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/madoxSio/go-cli-market/internal/httpapi"
)

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("serveur HTTP démarré", "addr", addr)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("arrêt du serveur", "err", err)
		os.Exit(1)
	}
}
