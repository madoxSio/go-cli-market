package store

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/madoxSio/go-cli-market/internal/market"
)

// testCategory renvoie une catégorie propre à ce test et supprime ses produits à la fin.
// Les recherches filtrent toujours sur elle, pour ignorer les produits déjà en base.
func testCategory(t *testing.T, m *Market) string {
	t.Helper()
	category := fmt.Sprintf("test-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		// Les lignes de panier qui pointent vers ces produits empêcheraient leur suppression.
		_, _ = m.db.Exec(`DELETE FROM cart_items WHERE product_id IN (SELECT id FROM products WHERE category = $1)`, category)
		_, _ = m.db.Exec(`DELETE FROM products WHERE category = $1`, category)
	})
	return category
}

func mustCreateProduct(t *testing.T, m *Market, in market.ProductInput) market.Product {
	t.Helper()
	p, err := in.Validate()
	if err != nil {
		t.Fatal(err)
	}
	created, err := m.CreateProduct(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	return created
}

func cents(c market.Cents) *market.Cents { return &c }

func TestCreateAndGetProduct(t *testing.T) {
	m := NewMarket(openTestDB(t))
	category := testCategory(t, m)

	created := mustCreateProduct(t, m, market.ProductInput{
		Name: "Clavier", Description: "Mécanique", Category: category, PriceHT: 4999,
	})
	if !market.IsBusinessID(created.Ref, market.PrefixProduct) {
		t.Errorf("ref = %q, attendu PDT-XXXXXX", created.Ref)
	}
	if created.ID == 0 || created.CreatedAt.IsZero() || created.PriceTTC != 5999 {
		t.Errorf("produit créé = %+v", created)
	}

	got, err := m.GetProduct(context.Background(), created.Ref)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Clavier" || got.PriceTTC != 5999 || got.VATRate != market.VATStandard {
		t.Errorf("produit relu = %+v", got)
	}

	if _, err := m.GetProduct(context.Background(), "PDT-ZZZZZZ"); !errors.Is(err, market.ErrNotFound) {
		t.Errorf("produit absent : erreur = %v, attendu ErrNotFound", err)
	}
}

func TestCreateProductRetriesOnRefCollision(t *testing.T) {
	m := NewMarket(openTestDB(t))
	category := testCategory(t, m)
	first := mustCreateProduct(t, m, market.ProductInput{Name: "A", Category: category, PriceHT: 100})

	// Le générateur renvoie d'abord une ref déjà prise, puis une ref libre.
	calls := 0
	newRef = func(prefix string) string {
		calls++
		if calls == 1 {
			return first.Ref
		}
		return market.NewBusinessID(prefix)
	}
	t.Cleanup(func() { newRef = market.NewBusinessID })

	second := mustCreateProduct(t, m, market.ProductInput{Name: "B", Category: category, PriceHT: 100})
	if calls != 2 || second.Ref == first.Ref {
		t.Errorf("appels = %d, ref = %s (première : %s)", calls, second.Ref, first.Ref)
	}
}

func TestSearchProducts(t *testing.T) {
	m := NewMarket(openTestDB(t))
	category := testCategory(t, m)
	reduced := market.VATReduced

	mustCreateProduct(t, m, market.ProductInput{Name: "Clavier mécanique", Description: "Switchs rouges", Category: category, PriceHT: 5000})                      // TTC 6000
	mustCreateProduct(t, m, market.ProductInput{Name: "Souris", Description: "Sans fil, compatible CLAVIER", Category: category, PriceHT: 2000})                   // TTC 2400
	mustCreateProduct(t, m, market.ProductInput{Name: "Livre Go", Description: "Apprendre 100% du langage", Category: category, PriceHT: 3000, VATRate: &reduced}) // TTC 3165

	tests := []struct {
		name   string
		filter market.ProductFilter
		want   []string
	}{
		{"tout", market.ProductFilter{}, []string{"Clavier mécanique", "Livre Go", "Souris"}},
		{"texte dans le nom ou la description, sans casse", market.ProductFilter{Query: "clavier"}, []string{"Clavier mécanique", "Souris"}},
		{"texte avec %", market.ProductFilter{Query: "100%"}, []string{"Livre Go"}},
		{"% seul ne joue pas le joker", market.ProductFilter{Query: "%"}, []string{"Livre Go"}},
		{"prix HT minimum", market.ProductFilter{MinHT: cents(3000)}, []string{"Clavier mécanique", "Livre Go"}},
		{"prix HT maximum", market.ProductFilter{MaxHT: cents(2999)}, []string{"Souris"}},
		{"fourchette TTC", market.ProductFilter{MinTTC: cents(2400), MaxTTC: cents(3165)}, []string{"Livre Go", "Souris"}},
		{"TTC juste sous la borne", market.ProductFilter{MaxTTC: cents(3164)}, []string{"Souris"}},
		{"texte et prix combinés", market.ProductFilter{Query: "clavier", MaxTTC: cents(5000)}, []string{"Souris"}},
		{"aucun résultat", market.ProductFilter{Query: "introuvable"}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.filter
			f.Category = category
			got, err := m.SearchProducts(context.Background(), f)
			if err != nil {
				t.Fatal(err)
			}
			if got == nil {
				t.Fatal("résultat nil : le JSON serait null au lieu de []")
			}
			names := make([]string, len(got))
			for i, p := range got {
				names[i] = p.Name
			}
			if fmt.Sprint(names) != fmt.Sprint(tt.want) {
				t.Errorf("résultats = %v, attendu %v", names, tt.want)
			}
		})
	}
}

func TestSearchProductsCategoryIgnoresCase(t *testing.T) {
	m := NewMarket(openTestDB(t))
	category := testCategory(t, m)
	mustCreateProduct(t, m, market.ProductInput{Name: "A", Category: category, PriceHT: 100})

	got, err := m.SearchProducts(context.Background(), market.ProductFilter{Category: "  " + category + "  "})
	if err != nil || len(got) != 1 {
		t.Fatalf("résultats = %v, err = %v", got, err)
	}
}
