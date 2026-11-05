package httpapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/madoxSio/go-cli-market/internal/auth"
	"github.com/madoxSio/go-cli-market/internal/store"
)

// shopTest prépare un routeur branché sur la base de test, avec un compte client
// et un compte admin propres au test, déjà connectés.
type shopTest struct {
	t           *testing.T
	h           http.Handler
	db          *sql.DB
	svc         *auth.Service
	adminToken  string
	clientToken string
	clientID    int64
	suffix      string // rend uniques les données créées par ce test
}

func newShopTest(t *testing.T) *shopTest {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL non défini")
	}
	db, err := store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := store.Migrate(db); err != nil {
		t.Fatal(err)
	}
	svc, err := auth.NewService(store.NewRepository(db))
	if err != nil {
		t.Fatal(err)
	}

	st := &shopTest{t: t, h: NewRouter(svc, store.NewMarket(db)), db: db, svc: svc,
		suffix: fmt.Sprintf("%d", time.Now().UnixNano())}
	st.adminToken = st.login("admin")
	st.clientToken = st.login("client")
	return st
}

// login crée un compte confirmé du rôle demandé, le supprime à la fin du test et renvoie son jeton.
func (st *shopTest) login(role string) string {
	st.t.Helper()
	email := fmt.Sprintf("%s-%s@test.local", role, st.suffix)
	user, err := st.svc.CreateUser(st.t.Context(), email, "motdepasse", role, true)
	if err != nil {
		st.t.Fatal(err)
	}
	st.t.Cleanup(func() { _, _ = st.db.Exec(`DELETE FROM users WHERE id = $1`, user.ID) })
	if role == "client" {
		st.clientID = user.ID
	}
	token, _, err := st.svc.Login(st.t.Context(), email, "motdepasse")
	if err != nil {
		st.t.Fatal(err)
	}
	return token
}

// call envoie une requête au routeur. body est encodé en JSON s'il n'est pas nil.
func (st *shopTest) call(method, path, token string, body any) *httptest.ResponseRecorder {
	st.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			st.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	st.h.ServeHTTP(rec, req)
	return rec
}

// expect vérifie le code HTTP et décode la réponse dans dst (si dst n'est pas nil).
func (st *shopTest) expect(rec *httptest.ResponseRecorder, status int, dst any) {
	st.t.Helper()
	if rec.Code != status {
		st.t.Fatalf("code = %d, attendu %d ; corps : %s", rec.Code, status, rec.Body.String())
	}
	if dst != nil {
		if err := json.NewDecoder(rec.Body).Decode(dst); err != nil {
			st.t.Fatalf("JSON invalide : %v", err)
		}
	}
}
