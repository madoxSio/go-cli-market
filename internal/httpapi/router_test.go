package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/madoxSio/go-cli-market/internal/auth"
)

func testRouter(t *testing.T) http.Handler {
	t.Helper()
	svc, err := auth.NewService(auth.NewMemoryRepository())
	if err != nil {
		t.Fatal(err)
	}
	return NewRouter(svc)
}

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	testRouter(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, attendu %d", rec.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("réponse JSON invalide : %v", err)
	}
	if got, want := body["status"], "ok"; got != want {
		t.Errorf("status = %q, attendu %q", got, want)
	}

	if got, want := rec.Header().Get("Content-Type"), "application/json"; got != want {
		t.Errorf("Content-Type = %q, attendu %q", got, want)
	}
}

func TestHealthRejectsPost(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rec := httptest.NewRecorder()

	testRouter(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code = %d, attendu %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
