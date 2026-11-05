package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/madoxSio/go-cli-market/internal/auth"
)

func TestAdminUserRoutes(t *testing.T) {
	svc, err := auth.NewService(auth.NewMemoryRepository())
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.EnsureAdmin(t.Context(), "admin@market.local", "adminadmin"); err != nil {
		t.Fatal(err)
	}
	h := NewRouter(svc, nil)

	denied := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	deniedRec := httptest.NewRecorder()
	h.ServeHTTP(deniedRec, denied)
	if deniedRec.Code != http.StatusUnauthorized {
		t.Fatalf("sans jeton = %d", deniedRec.Code)
	}

	logged := postJSON(t, h, "/auth/login", "", map[string]string{
		"email": "admin@market.local", "password": "adminadmin",
	})
	if logged.Code != http.StatusOK {
		t.Fatalf("login admin = %d %s", logged.Code, logged.Body.String())
	}
	var session struct {
		Token string `json:"token"`
	}
	decode(t, logged, &session)

	created := postJSON(t, h, "/admin/users", session.Token, map[string]any{
		"email": "ada@example.com", "password": "motdepasse", "role": "client", "confirmed": false,
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("création = %d %s", created.Code, created.Body.String())
	}
	var user auth.User
	decode(t, created, &user)

	confirmed := postJSON(t, h, "/admin/users/"+strconv.FormatInt(user.ID, 10)+"/confirm", session.Token, map[string]any{})
	if confirmed.Code != http.StatusOK {
		t.Fatalf("confirm = %d %s", confirmed.Code, confirmed.Body.String())
	}

	clientLogin := postJSON(t, h, "/auth/login", "", map[string]string{
		"email": "ada@example.com", "password": "motdepasse",
	})
	if clientLogin.Code != http.StatusOK {
		t.Fatalf("login client = %d %s", clientLogin.Code, clientLogin.Body.String())
	}
	var clientSession struct {
		Token string `json:"token"`
	}
	decode(t, clientLogin, &clientSession)
	forbidden := postJSON(t, h, "/admin/users", clientSession.Token, map[string]any{
		"email": "x@example.com", "password": "motdepasse",
	})
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("client sur l'admin = %d", forbidden.Code)
	}
}
