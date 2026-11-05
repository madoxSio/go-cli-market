package httpapi

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/madoxSio/go-cli-market/internal/market"
)

// listProducts répond à GET /products?q=&category=&min_price=&max_price=&min_ttc=&max_ttc=.
func (h *shopHandler) listProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	f := market.ProductFilter{
		Query:    query.Get("q"),
		Category: query.Get("category"),
	}

	// Chaque paramètre de prix est facultatif ; présent, il doit être un nombre de centimes positif.
	var err error
	if f.MinHT, err = centsParam(query, "min_price"); err != nil {
		writeError(w, err)
		return
	}
	if f.MaxHT, err = centsParam(query, "max_price"); err != nil {
		writeError(w, err)
		return
	}
	if f.MinTTC, err = centsParam(query, "min_ttc"); err != nil {
		writeError(w, err)
		return
	}
	if f.MaxTTC, err = centsParam(query, "max_ttc"); err != nil {
		writeError(w, err)
		return
	}

	products, err := h.shop.SearchProducts(r.Context(), f)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJson(w, http.StatusOK, products)
}

// getProduct répond à GET /products/{ref}.
func (h *shopHandler) getProduct(w http.ResponseWriter, r *http.Request) {
	ref := r.PathValue("ref")
	if !market.IsBusinessID(ref, market.PrefixProduct) {
		writeError(w, fmt.Errorf("produit %s : %w", ref, market.ErrNotFound))
		return
	}
	p, err := h.shop.GetProduct(r.Context(), ref)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJson(w, http.StatusOK, p)
}

// createProduct répond à POST /admin/products.
func (h *shopHandler) createProduct(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	var in market.ProductInput
	if !readJSON(w, r, &in) {
		return
	}
	p, err := in.Validate()
	if err != nil {
		writeError(w, err)
		return
	}
	created, err := h.shop.CreateProduct(r.Context(), p)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJson(w, http.StatusCreated, created)
}

// centsParam lit un paramètre d'URL facultatif exprimé en centimes.
// Absent : nil, sans erreur. Présent mais invalide : market.ErrInvalidInput.
func centsParam(query url.Values, name string) (*market.Cents, error) {
	raw := query.Get(name)
	if raw == "" {
		return nil, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return nil, fmt.Errorf("%s doit être un nombre entier de centimes positif : %w", name, market.ErrInvalidInput)
	}
	c := market.Cents(n)
	return &c, nil
}
