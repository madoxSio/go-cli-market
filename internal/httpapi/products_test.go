package httpapi

import (
	"net/http"
	"testing"

	"github.com/madoxSio/go-cli-market/internal/market"
)

func TestProductRoutes(t *testing.T) {
	st := newShopTest(t)
	category := "test-" + st.suffix
	t.Cleanup(func() { _, _ = st.db.Exec(`DELETE FROM products WHERE category = $1`, category) })

	input := map[string]any{"name": "Clavier", "description": "Mécanique", "category": category, "price_ht": 4999}

	// Créer un produit : réservé aux admins.
	st.expect(st.call(http.MethodPost, "/admin/products", "", input), http.StatusUnauthorized, nil)
	st.expect(st.call(http.MethodPost, "/admin/products", st.clientToken, input), http.StatusForbidden, nil)

	var created market.Product
	st.expect(st.call(http.MethodPost, "/admin/products", st.adminToken, input), http.StatusCreated, &created)
	if !market.IsBusinessID(created.Ref, market.PrefixProduct) || created.PriceTTC != 5999 {
		t.Fatalf("produit créé = %+v", created)
	}

	// Données invalides → 400.
	bad := map[string]any{"name": "", "category": category, "price_ht": 100}
	st.expect(st.call(http.MethodPost, "/admin/products", st.adminToken, bad), http.StatusBadRequest, nil)

	// Lire un produit : public.
	var got market.Product
	st.expect(st.call(http.MethodGet, "/products/"+created.Ref, "", nil), http.StatusOK, &got)
	if got.Name != "Clavier" {
		t.Errorf("produit lu = %+v", got)
	}
	st.expect(st.call(http.MethodGet, "/products/PDT-ZZZZZZ", "", nil), http.StatusNotFound, nil)
	st.expect(st.call(http.MethodGet, "/products/n-importe-quoi", "", nil), http.StatusNotFound, nil)

	// Rechercher : public, avec filtres.
	var found []market.Product
	st.expect(st.call(http.MethodGet, "/products?category="+category+"&q=clav&max_ttc=5999", "", nil), http.StatusOK, &found)
	if len(found) != 1 || found[0].Ref != created.Ref {
		t.Errorf("recherche = %+v", found)
	}
	st.expect(st.call(http.MethodGet, "/products?category="+category+"&max_ttc=5998", "", nil), http.StatusOK, &found)
	if len(found) != 0 {
		t.Errorf("recherche sous le prix = %+v, attendu aucun résultat", found)
	}

	// Paramètre de prix invalide → 400.
	st.expect(st.call(http.MethodGet, "/products?min_price=abc", "", nil), http.StatusBadRequest, nil)
	st.expect(st.call(http.MethodGet, "/products?max_ttc=-1", "", nil), http.StatusBadRequest, nil)
}
