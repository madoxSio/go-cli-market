package httpapi

import (
	"net/http"
	"time"

	"github.com/madoxSio/go-cli-market/internal/auth"
	"github.com/madoxSio/go-cli-market/internal/store"
)

// shopHandler regroupe les routes du catalogue, du panier et des commandes.
type shopHandler struct {
	svc  *auth.Service
	shop *store.Market
	now  func() time.Time
}

// requireUser renvoie l'utilisateur connecté. Sinon, la réponse 401 est déjà écrite et ok vaut false.
func (h *shopHandler) requireUser(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	user, err := h.svc.CurrentUser(r.Context(), bearerToken(r))
	if err != nil {
		writeAuthError(w, err)
		return auth.User{}, false
	}
	return user, true
}

// requireAdmin renvoie l'administrateur connecté. Sinon, la réponse 401 ou 403 est déjà écrite.
func (h *shopHandler) requireAdmin(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return auth.User{}, false
	}
	if user.Role != "admin" {
		writeAuthError(w, auth.ErrForbidden)
		return auth.User{}, false
	}
	return user, true
}
