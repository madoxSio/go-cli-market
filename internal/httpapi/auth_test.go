package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func postJSON(t *testing.T, h http.Handler, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(dst); err != nil {
		t.Fatalf("json : %v (corps %s)", err, rec.Body.String())
	}
}

func TestAuthHTTPFlow(t *testing.T) {
	h := testRouter(t)
	const email = "client@example.com"

	created := postJSON(t, h, "/auth/register", "", map[string]string{
		"email": email, "password": "motdepasse",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("register = %d %s", created.Code, created.Body.String())
	}
	var registered codeResponse
	decode(t, created, &registered)
	if registered.ConfirmationCode == "" {
		t.Fatal("code de confirmation absent")
	}

	denied := postJSON(t, h, "/auth/login", "", map[string]string{
		"email": email, "password": "motdepasse",
	})
	if denied.Code != http.StatusForbidden {
		t.Fatalf("login non confirmé = %d", denied.Code)
	}

	confirmed := postJSON(t, h, "/auth/confirm", "", map[string]string{
		"email": email, "code": registered.ConfirmationCode,
	})
	if confirmed.Code != http.StatusOK {
		t.Fatalf("confirm = %d %s", confirmed.Code, confirmed.Body.String())
	}

	logged := postJSON(t, h, "/auth/login", "", map[string]string{
		"email": email, "password": "motdepasse",
	})
	if logged.Code != http.StatusOK {
		t.Fatalf("login = %d %s", logged.Code, logged.Body.String())
	}
	var session struct {
		Token string `json:"token"`
		User  struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	decode(t, logged, &session)

	me := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	me.Header.Set("Authorization", "Bearer "+session.Token)
	meRec := httptest.NewRecorder()
	h.ServeHTTP(meRec, me)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me = %d", meRec.Code)
	}

	forgot := postJSON(t, h, "/auth/forgot-password", "", map[string]string{"email": email})
	if forgot.Code != http.StatusOK {
		t.Fatalf("forgot = %d", forgot.Code)
	}
	var reset codeResponse
	decode(t, forgot, &reset)
	if reset.ResetCode == "" {
		t.Fatal("code de réinitialisation absent")
	}

	updated := postJSON(t, h, "/auth/reset-password", "", map[string]string{
		"email": email, "code": reset.ResetCode, "new_password": "autre-mot-de-passe",
	})
	if updated.Code != http.StatusOK {
		t.Fatalf("reset = %d %s", updated.Code, updated.Body.String())
	}

	meRec = httptest.NewRecorder()
	h.ServeHTTP(meRec, me)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("session après reset = %d, attendu 401", meRec.Code)
	}

	again := postJSON(t, h, "/auth/login", "", map[string]string{
		"email": email, "password": "autre-mot-de-passe",
	})
	if again.Code != http.StatusOK {
		t.Fatalf("login nouveau mot de passe = %d %s", again.Code, again.Body.String())
	}

	duplicate := postJSON(t, h, "/auth/register", "", map[string]string{
		"email": email, "password": "motdepasse",
	})
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("doublon = %d", duplicate.Code)
	}

	unknown := postJSON(t, h, "/auth/forgot-password", "", map[string]string{"email": "inconnu@example.com"})
	if unknown.Code != http.StatusOK {
		t.Fatalf("email inconnu = %d", unknown.Code)
	}
	var hidden codeResponse
	decode(t, unknown, &hidden)
	if hidden.ResetCode != "" {
		t.Fatal("un code a été renvoyé pour un email inconnu")
	}
}
