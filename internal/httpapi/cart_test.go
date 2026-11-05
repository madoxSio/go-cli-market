package httpapi

import (
	"net/http"
	"testing"

	"github.com/madoxSio/go-cli-market/internal/market"
)

// createTestProduct ajoute un produit par l'API admin, dans une catégorie propre au test.
func (st *shopTest) createTestProduct(name string, priceHT int) market.Product {
	st.t.Helper()
	category := "test-" + st.suffix
	st.t.Cleanup(func() {
		_, _ = st.db.Exec(`DELETE FROM cart_items WHERE product_id IN (SELECT id FROM products WHERE category = $1)`, category)
		_, _ = st.db.Exec(`DELETE FROM products WHERE category = $1`, category)
	})
	var p market.Product
	st.expect(st.call(http.MethodPost, "/admin/products", st.adminToken,
		map[string]any{"name": name, "category": category, "price_ht": priceHT}), http.StatusCreated, &p)
	return p
}

func items(pairs ...any) map[string]any {
	var list []map[string]any
	for i := 0; i+1 < len(pairs); i += 2 {
		list = append(list, map[string]any{"product_ref": pairs[i], "quantity": pairs[i+1]})
	}
	return map[string]any{"items": list}
}

func TestCartRoutes(t *testing.T) {
	st := newShopTest(t)
	keyboard := st.createTestProduct("Clavier", 5000) // 60,00 € TTC
	mouse := st.createTestProduct("Souris", 2000)     // 24,00 € TTC

	// Sans jeton : 401 partout.
	st.expect(st.call(http.MethodGet, "/cart", "", nil), http.StatusUnauthorized, nil)
	st.expect(st.call(http.MethodPost, "/cart/items", "", items(keyboard.Ref, 1)), http.StatusUnauthorized, nil)

	// Panier vide au départ.
	var cart market.Cart
	st.expect(st.call(http.MethodGet, "/cart", st.clientToken, nil), http.StatusOK, &cart)
	if cart.Ref != "" || len(cart.Items) != 0 {
		t.Fatalf("panier initial = %+v", cart)
	}

	// Ajouter deux produits en une requête.
	st.expect(st.call(http.MethodPost, "/cart/items", st.clientToken, items(keyboard.Ref, 1, mouse.Ref, 2)), http.StatusOK, &cart)
	if len(cart.Items) != 2 || cart.TotalTTC != 6000+4800 || cart.Shipping != 0 {
		t.Fatalf("après ajout = %+v", cart)
	}

	// Modifier : clavier à 3, souris retirée.
	st.expect(st.call(http.MethodPatch, "/cart/items", st.clientToken, items(keyboard.Ref, 3, mouse.Ref, 0)), http.StatusOK, &cart)
	if len(cart.Items) != 1 || cart.Items[0].Quantity != 3 || cart.TotalTTC != 18000 {
		t.Fatalf("après modification = %+v", cart)
	}

	// Supprimer le clavier.
	st.expect(st.call(http.MethodDelete, "/cart/items/"+keyboard.Ref, st.clientToken, nil), http.StatusOK, &cart)
	if len(cart.Items) != 0 || cart.TotalTTC != 0 {
		t.Fatalf("après suppression = %+v", cart)
	}

	// Erreurs.
	st.expect(st.call(http.MethodPost, "/cart/items", st.clientToken, items()), http.StatusBadRequest, nil)                  // liste vide
	st.expect(st.call(http.MethodPost, "/cart/items", st.clientToken, items(keyboard.Ref, 0)), http.StatusBadRequest, nil)   // 0 interdit en ajout
	st.expect(st.call(http.MethodPost, "/cart/items", st.clientToken, items(keyboard.Ref, 100)), http.StatusBadRequest, nil) // > 99
	st.expect(st.call(http.MethodPost, "/cart/items", st.clientToken, items("PDT-ZZZZZZ", 1)), http.StatusNotFound, nil)     // inconnu
	st.expect(st.call(http.MethodPost, "/cart/items", st.clientToken, items("pas-une-ref", 1)), http.StatusNotFound, nil)    // mal formé
	st.expect(st.call(http.MethodDelete, "/cart/items/"+mouse.Ref, st.clientToken, nil), http.StatusNotFound, nil)           // absent du panier
	st.expect(st.call(http.MethodPatch, "/cart/items", st.clientToken, items(keyboard.Ref, -1)), http.StatusBadRequest, nil) // négatif
}
