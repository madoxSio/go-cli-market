package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

//writejson envoie c encodé en JSON, avec le code status http
func writeJson(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("Encodage JSON de la réponse", "err", err)
	}
}