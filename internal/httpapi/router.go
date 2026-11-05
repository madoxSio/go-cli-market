// Package httpapi expose l'API HTTP de la plateforme.
package httpapi

import (
	"net/http"
	"time"

	"github.com/madoxSio/go-cli-market/internal/auth"
	"github.com/madoxSio/go-cli-market/internal/store"
)

// NewRouter construit le routeur de l'API et y enregistre toutes les routes.
// svc gère les comptes et les sessions, shop le catalogue, les paniers et les commandes.
func NewRouter(svc *auth.Service, shop *store.Market) http.Handler {
	authH := &authHandler{svc: svc}
	shopH := &shopHandler{svc: svc, shop: shop, now: time.Now}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /auth/register", authH.register)
	mux.HandleFunc("POST /auth/confirm", authH.confirm)
	mux.HandleFunc("POST /auth/confirmation-code", authH.renewConfirmation)
	mux.HandleFunc("POST /auth/login", authH.login)
	mux.HandleFunc("POST /auth/forgot-password", authH.forgotPassword)
	mux.HandleFunc("POST /auth/reset-password", authH.resetPassword)
	mux.HandleFunc("POST /auth/logout", authH.logout)
	mux.HandleFunc("GET /auth/me", authH.me)
	mux.HandleFunc("GET /admin/users", authH.listUsers)
	mux.HandleFunc("POST /admin/users", authH.createUser)
	mux.HandleFunc("PUT /admin/users/{id}", authH.updateUser)
	mux.HandleFunc("DELETE /admin/users/{id}", authH.deleteUser)
	mux.HandleFunc("POST /admin/users/{id}/confirm", authH.confirmUser)

	mux.HandleFunc("GET /products", shopH.listProducts)
	mux.HandleFunc("GET /products/{ref}", shopH.getProduct)
	mux.HandleFunc("POST /admin/products", shopH.createProduct)

	mux.HandleFunc("GET /cart", shopH.getCart)
	mux.HandleFunc("POST /cart/items", shopH.addCartItems)
	mux.HandleFunc("PATCH /cart/items", shopH.setCartItems)
	mux.HandleFunc("DELETE /cart/items/{productRef}", shopH.removeCartItem)
	return mux
}

// handleHealth indique que le serveur est en vie.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJson(w, http.StatusOK, map[string]string{"status": "ok"})
}
