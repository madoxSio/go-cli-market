package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/madoxSio/go-cli-market/internal/market"
)

func TestWriteError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{"introuvable", market.ErrNotFound, http.StatusNotFound, "introuvable"},
		{"introuvable emballée", fmt.Errorf("produit PDT-7D2K8N : %w", market.ErrNotFound), http.StatusNotFound, "produit PDT-7D2K8N : introuvable"},
		{"invalide", market.ErrInvalidInput, http.StatusBadRequest, "données invalides"},
		{"non connecté", market.ErrUnauthorized, http.StatusUnauthorized, "authentification requise"},
		{"interdit", market.ErrForbidden, http.StatusForbidden, "accès interdit"},
		{"conflit", market.ErrConflict, http.StatusConflict, "conflit avec l'état actuel"},
		{"imprévue", errors.New("connexion à la base perdue"), http.StatusInternalServerError, "erreur interne du serveur"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			writeError(rec, tt.err)

			if rec.Code != tt.wantStatus {
				t.Errorf("code = %d, attendu %d", rec.Code, tt.wantStatus)
			}
			var body map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("réponse JSON invalide : %v", err)
			}
			if got := body["error"]; got != tt.wantMsg {
				t.Errorf("error = %q, attendu %q", got, tt.wantMsg)
			}
		})
	}
}