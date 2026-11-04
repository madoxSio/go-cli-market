package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
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
