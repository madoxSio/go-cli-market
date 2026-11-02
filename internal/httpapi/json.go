package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/madoxSio/go-cli-market/internal/market"
)

// writeJSON envoie v encodé en JSON, avec le code HTTP status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encodage JSON de la réponse", "err", err)
	}
}

// writeError traduit une erreur métier en réponse JSON avec le bon code HTTP.
func writeError(w http.ResponseWriter, err error) {
	var status int
	switch {
	case errors.Is(err, market.ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, market.ErrUnauthorized):
		status = http.StatusUnauthorized
	case errors.Is(err, market.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, market.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, market.ErrConflict):
		status = http.StatusConflict
	default:
		// Erreur imprévue : on la journalise, mais on n'en montre pas le détail au client.
		slog.Error("erreur interne", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "erreur interne du serveur"})
		return
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}