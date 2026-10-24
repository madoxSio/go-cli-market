// Package httpapi expose l'API HTTP de la plateforme.
package httpapi

import "net/http"

// NewRouter construit le routeur de l'API et y enregistre toutes les routes.
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	return mux
}

// handleHealth indique que le serveur est en vie.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
