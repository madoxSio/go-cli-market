package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/madoxSio/go-cli-market/internal/market"
)

// writeJson envoie v encodé en JSON, avec le code de statut HTTP.
func writeJson(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encodage JSON de la réponse", "err", err)
	}
}

// readJSON décode le corps de la requête. En cas d'échec, la réponse 400 est déjà écrite.
func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeJson(w, http.StatusBadRequest, map[string]string{"error": "JSON invalide"})
		return false
	}
	return true
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
		writeJson(w, http.StatusInternalServerError, map[string]string{"error": "erreur interne du serveur"})
		return
	}
	writeJson(w, status, map[string]string{"error": err.Error()})
}
