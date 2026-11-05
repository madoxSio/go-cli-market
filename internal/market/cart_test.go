package market

import (
	"errors"
	"testing"
)

func TestNewItem(t *testing.T) {
	p := Product{Ref: "PDT-AAAAAA", Name: "Livre", PriceHT: 999, VATRate: VATReduced}

	it := NewItem(p, 3)

	if it.ProductRef != "PDT-AAAAAA" || it.Name != "Livre" || it.Quantity != 3 {
		t.Errorf("ligne = %+v", it)
	}
	// 9,99 € à 5,5 % = 10,539 € → 10,54 € par unité ; 3 × 10,54 € = 31,62 €.
	if it.UnitPriceTTC != 1054 {
		t.Errorf("prix unitaire TTC = %d, attendu 1054", it.UnitPriceTTC)
	}
	if it.LineTTC != 3162 {
		t.Errorf("total de la ligne = %d, attendu 3162", it.LineTTC)
	}
}

func TestTotal(t *testing.T) {
	items := []Item{
		NewItem(Product{PriceHT: 1000, VATRate: VATStandard}, 2), // 2 × 12,00 €
		NewItem(Product{PriceHT: 999, VATRate: VATReduced}, 3),   // 3 × 10,54 €
	}
	if got := Total(items); got != 2400+3162 {
		t.Errorf("Total = %d, attendu %d", got, 2400+3162)
	}
	if got := Total(nil); got != 0 {
		t.Errorf("Total(panier vide) = %d, attendu 0", got)
	}
}

func TestValidateQuantity(t *testing.T) {
	for _, q := range []int{1, 50, MaxQuantity} {
		if err := ValidateQuantity(q); err != nil {
			t.Errorf("ValidateQuantity(%d) = %v, attendu nil", q, err)
		}
	}
	for _, q := range []int{-1, 0, MaxQuantity + 1} {
		if err := ValidateQuantity(q); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("ValidateQuantity(%d) = %v, attendu ErrInvalidInput", q, err)
		}
	}
}
