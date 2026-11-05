package httpapi

import (
	"fmt"
	"net/http"

	"github.com/madoxSio/go-cli-market/internal/api"
	"github.com/madoxSio/go-cli-market/internal/market"
)

// getCart répond à GET /cart.
func (h *shopHandler) getCart(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	cart, err := h.shop.GetCart(r.Context(), user.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJson(w, http.StatusOK, cart)
}

// addCartItems répond à POST /cart/items : ajoute un ou plusieurs produits.
func (h *shopHandler) addCartItems(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	items, ok := readCartItems(w, r, false)
	if !ok {
		return
	}
	cart, err := h.shop.AddItems(r.Context(), user.ID, items)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJson(w, http.StatusOK, cart)
}

// setCartItems répond à PATCH /cart/items : remplace les quantités, 0 retire le produit.
func (h *shopHandler) setCartItems(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	items, ok := readCartItems(w, r, true)
	if !ok {
		return
	}
	cart, err := h.shop.SetQuantities(r.Context(), user.ID, items)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJson(w, http.StatusOK, cart)
}

// removeCartItem répond à DELETE /cart/items/{productRef}.
func (h *shopHandler) removeCartItem(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	cart, err := h.shop.RemoveItem(r.Context(), user.ID, r.PathValue("productRef"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJson(w, http.StatusOK, cart)
}

// readCartItems lit un api.CartItemsRequest et vérifie chaque ligne : une référence de produit
// bien formée et une quantité de 1 à market.MaxQuantity (ou 0 si allowZero, pour retirer).
// En cas de problème, la réponse d'erreur est déjà écrite et ok vaut false.
func readCartItems(w http.ResponseWriter, r *http.Request, allowZero bool) ([]api.ItemRequest, bool) {
	var req api.CartItemsRequest
	if !readJSON(w, r, &req) {
		return nil, false
	}
	if len(req.Items) == 0 {
		writeError(w, fmt.Errorf("la liste items est vide : %w", market.ErrInvalidInput))
		return nil, false
	}
	for _, it := range req.Items {
		if !market.IsBusinessID(it.ProductRef, market.PrefixProduct) {
			writeError(w, fmt.Errorf("produit %q : %w", it.ProductRef, market.ErrNotFound))
			return nil, false
		}
		if it.Quantity == 0 && allowZero {
			continue // PATCH : 0 veut dire « retirer »
		}
		if err := market.ValidateQuantity(it.Quantity); err != nil {
			writeError(w, err)
			return nil, false
		}
	}
	return req.Items, true
}
