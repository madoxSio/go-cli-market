package httpapi

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/madoxSio/go-cli-market/internal/market"
)

var testCard = map[string]any{"card": map[string]string{"number": "4242 4242 4242 4242", "expiry": "12/30", "cvc": "123"}}

func TestCheckoutAndClientOrders(t *testing.T) {
	st := newShopTest(t)
	keyboard := st.createTestProduct("Clavier", 5000) // 60,00 € TTC

	// Panier vide → 409.
	st.expect(st.call(http.MethodPost, "/cart/checkout", st.clientToken, testCard), http.StatusConflict, nil)

	st.expect(st.call(http.MethodPost, "/cart/items", st.clientToken, items(keyboard.Ref, 2)), http.StatusOK, nil)

	// Carte refusée → 400, et le panier n'est pas touché.
	badCard := map[string]any{"card": map[string]string{"number": "4242 4242 4242 4241", "expiry": "12/30", "cvc": "123"}}
	st.expect(st.call(http.MethodPost, "/cart/checkout", st.clientToken, badCard), http.StatusBadRequest, nil)
	expired := map[string]any{"card": map[string]string{"number": "4242424242424242", "expiry": "01/20", "cvc": "123"}}
	st.expect(st.call(http.MethodPost, "/cart/checkout", st.clientToken, expired), http.StatusBadRequest, nil)
	var cart market.Cart
	st.expect(st.call(http.MethodGet, "/cart", st.clientToken, nil), http.StatusOK, &cart)
	if len(cart.Items) != 1 {
		t.Fatalf("le panier ne doit pas changer après un refus : %+v", cart)
	}

	// Paiement accepté → 201, commande payée.
	var order market.Order
	st.expect(st.call(http.MethodPost, "/cart/checkout", st.clientToken, testCard), http.StatusCreated, &order)
	if order.Status != market.StatusPaid || order.TotalTTC != 12000 || order.CardLast4 != "4242" || len(order.Items) != 1 {
		t.Fatalf("commande = %+v", order)
	}

	// Le panier est vide ensuite.
	st.expect(st.call(http.MethodGet, "/cart", st.clientToken, nil), http.StatusOK, &cart)
	if len(cart.Items) != 0 {
		t.Errorf("panier après paiement = %+v", cart)
	}

	// Mes commandes.
	var orders []market.Order
	st.expect(st.call(http.MethodGet, "/orders", st.clientToken, nil), http.StatusOK, &orders)
	if len(orders) != 1 || orders[0].Ref != order.Ref {
		t.Fatalf("mes commandes = %+v", orders)
	}
	var got market.Order
	st.expect(st.call(http.MethodGet, "/orders/"+order.Ref, st.clientToken, nil), http.StatusOK, &got)
	if got.Ref != order.Ref {
		t.Errorf("commande lue = %+v", got)
	}
	st.expect(st.call(http.MethodGet, "/orders/CMD-ZZZZZZ", st.clientToken, nil), http.StatusNotFound, nil)
	st.expect(st.call(http.MethodGet, "/orders", "", nil), http.StatusUnauthorized, nil)

	// Un autre client ne voit pas cette commande (404, pas 403).
	otherEmail := fmt.Sprintf("autre-%d@test.local", time.Now().UnixNano())
	other, err := st.svc.CreateUser(t.Context(), otherEmail, "motdepasse", "client", true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = st.db.Exec(`DELETE FROM users WHERE id = $1`, other.ID) })
	otherToken, _, err := st.svc.Login(t.Context(), otherEmail, "motdepasse")
	if err != nil {
		t.Fatal(err)
	}
	st.expect(st.call(http.MethodGet, "/orders/"+order.Ref, otherToken, nil), http.StatusNotFound, nil)
}

func TestAdminOrders(t *testing.T) {
	st := newShopTest(t)
	book := st.createTestProduct("Livre", 1000) // 12,00 € TTC

	create := map[string]any{"user_id": st.clientID, "items": []map[string]any{{"product_ref": book.Ref, "quantity": 2}}}

	// Réservé aux admins.
	st.expect(st.call(http.MethodPost, "/admin/orders", st.clientToken, create), http.StatusForbidden, nil)
	st.expect(st.call(http.MethodGet, "/admin/orders", st.clientToken, nil), http.StatusForbidden, nil)

	// Créer une commande pour le client : pending.
	var order market.Order
	st.expect(st.call(http.MethodPost, "/admin/orders", st.adminToken, create), http.StatusCreated, &order)
	if order.Status != market.StatusPending || order.UserID != st.clientID || order.TotalTTC != 2400 {
		t.Fatalf("commande admin = %+v", order)
	}

	// Erreurs de création.
	st.expect(st.call(http.MethodPost, "/admin/orders", st.adminToken, map[string]any{"items": create["items"]}), http.StatusBadRequest, nil)
	st.expect(st.call(http.MethodPost, "/admin/orders", st.adminToken, map[string]any{"user_id": st.clientID}), http.StatusBadRequest, nil)
	st.expect(st.call(http.MethodPost, "/admin/orders", st.adminToken,
		map[string]any{"user_id": 999999999, "items": create["items"]}), http.StatusNotFound, nil)

	// Le client la voit dans ses commandes.
	var mine []market.Order
	st.expect(st.call(http.MethodGet, "/orders", st.clientToken, nil), http.StatusOK, &mine)
	if len(mine) != 1 || mine[0].Ref != order.Ref {
		t.Fatalf("commandes du client = %+v", mine)
	}

	// Changer le statut.
	path := "/admin/orders/" + order.Ref + "/status"
	st.expect(st.call(http.MethodPatch, path, st.adminToken, map[string]string{"status": "paid"}), http.StatusOK, &order)
	if order.Status != market.StatusPaid {
		t.Fatalf("statut = %s", order.Status)
	}
	st.expect(st.call(http.MethodPatch, path, st.adminToken, map[string]string{"status": "delivered"}), http.StatusConflict, nil)
	st.expect(st.call(http.MethodPatch, path, st.adminToken, map[string]string{"status": "perdue"}), http.StatusBadRequest, nil)
	st.expect(st.call(http.MethodPatch, path, st.adminToken, map[string]string{"status": "cancelled"}), http.StatusBadRequest, nil)
	st.expect(st.call(http.MethodPatch, path, st.adminToken,
		map[string]string{"status": "cancelled", "reason": "Rupture de stock"}), http.StatusOK, &order)
	if order.Status != market.StatusCancelled || order.CancelReason != "Rupture de stock" {
		t.Fatalf("annulation = %+v", order)
	}
	st.expect(st.call(http.MethodPatch, "/admin/orders/CMD-ZZZZZZ/status", st.adminToken, map[string]string{"status": "paid"}), http.StatusNotFound, nil)

	// Liste admin filtrée.
	var cancelled []market.Order
	st.expect(st.call(http.MethodGet, "/admin/orders?status=cancelled", st.adminToken, nil), http.StatusOK, &cancelled)
	found := false
	for _, o := range cancelled {
		found = found || o.Ref == order.Ref
		if o.Status != market.StatusCancelled {
			t.Errorf("commande %s au statut %s dans le filtre", o.Ref, o.Status)
		}
	}
	if !found {
		t.Error("la commande annulée est absente de la liste admin")
	}
	st.expect(st.call(http.MethodGet, "/admin/orders?status=perdue", st.adminToken, nil), http.StatusBadRequest, nil)
}
