package httpapi

import (
	"fmt"
	"net/http"

	"github.com/madoxSio/go-cli-market/internal/api"
	"github.com/madoxSio/go-cli-market/internal/market"
)

// checkout répond à POST /cart/checkout : vérifie la carte, puis paie le panier en cours.
func (h *shopHandler) checkout(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var req api.CheckoutRequest
	if !readJSON(w, r, &req) {
		return
	}
	last4, err := market.ValidateCard(req.Card, h.now())
	if err != nil {
		writeError(w, err)
		return
	}
	order, err := h.shop.Checkout(r.Context(), user.ID, last4)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJson(w, http.StatusCreated, order)
}

// listMyOrders répond à GET /orders : les commandes du client connecté.
func (h *shopHandler) listMyOrders(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	orders, err := h.shop.ListUserOrders(r.Context(), user.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJson(w, http.StatusOK, orders)
}

// getMyOrder répond à GET /orders/{ref}. La commande d'un autre client répond 404, comme
// une commande qui n'existe pas : on ne révèle pas qu'elle existe.
func (h *shopHandler) getMyOrder(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	ref := r.PathValue("ref")
	notFound := fmt.Errorf("commande %s : %w", ref, market.ErrNotFound)
	if !market.IsBusinessID(ref, market.PrefixOrder) {
		writeError(w, notFound)
		return
	}
	order, err := h.shop.GetOrder(r.Context(), ref)
	if err != nil {
		writeError(w, err)
		return
	}
	if order.UserID != user.ID {
		writeError(w, notFound)
		return
	}
	writeJson(w, http.StatusOK, order)
}

// listAllOrders répond à GET /admin/orders?status= : les commandes de tous les clients.
func (h *shopHandler) listAllOrders(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	status := market.OrderStatus(r.URL.Query().Get("status"))
	if status != "" && !status.Valid() {
		writeError(w, fmt.Errorf("statut %q inconnu : %w", status, market.ErrInvalidInput))
		return
	}
	orders, err := h.shop.ListOrders(r.Context(), status)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJson(w, http.StatusOK, orders)
}

// createOrder répond à POST /admin/orders : une commande « pending » pour un client.
func (h *shopHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	var req api.CreateOrderRequest
	if !readJSON(w, r, &req) {
		return
	}
	if req.UserID <= 0 {
		writeError(w, fmt.Errorf("user_id est obligatoire : %w", market.ErrInvalidInput))
		return
	}
	if len(req.Items) == 0 {
		writeError(w, fmt.Errorf("la liste items est vide : %w", market.ErrInvalidInput))
		return
	}
	for _, it := range req.Items {
		if !market.IsBusinessID(it.ProductRef, market.PrefixProduct) {
			writeError(w, fmt.Errorf("produit %q : %w", it.ProductRef, market.ErrNotFound))
			return
		}
		if err := market.ValidateQuantity(it.Quantity); err != nil {
			writeError(w, err)
			return
		}
	}
	order, err := h.shop.CreateOrder(r.Context(), req.UserID, req.Items)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJson(w, http.StatusCreated, order)
}

// updateOrderStatus répond à PATCH /admin/orders/{ref}/status.
func (h *shopHandler) updateOrderStatus(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	ref := r.PathValue("ref")
	if !market.IsBusinessID(ref, market.PrefixOrder) {
		writeError(w, fmt.Errorf("commande %s : %w", ref, market.ErrNotFound))
		return
	}
	var req api.UpdateStatusRequest
	if !readJSON(w, r, &req) {
		return
	}
	order, err := h.shop.UpdateOrderStatus(r.Context(), ref, req.Status, req.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJson(w, http.StatusOK, order)
}
