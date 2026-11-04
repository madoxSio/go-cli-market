package market

import "testing"

func TestPriceWithVAT(t *testing.T) {
	tests := []struct {
		name string
		ht   Cents
		rate BasisPoints
		want Cents
	}{
		{"prix nul", 0, VATStandard, 0},
		{"taux nul", 1234, 0, 1234},
		{"20 % exact", 1000, VATStandard, 1200},
		{"10 % exact", 1000, VATIntermediate, 1100},
		{"5,5 % exact", 1000, VATReduced, 1055},
		{"2,1 % exact", 1000, VATSuper, 1021},
		{"arrondi vers le haut", 999, VATStandard, 1199},     // 11,988 € → 11,99 €
		{"arrondi vers le bas", 1, VATReduced, 1},            // 0,01055 € → 0,01 €
		{"demi-centime vers le haut", 5, VATIntermediate, 6}, // 0,055 € → 0,06 €
		{"5,5 % arrondi", 10, VATReduced, 11},                // 0,1055 € → 0,11 €
		{"gros montant", 99999999, VATStandard, 119999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PriceWithVAT(tt.ht, tt.rate); got != tt.want {
				t.Errorf("PriceWithVAT(%d, %d) = %d, attendu %d", tt.ht, tt.rate, got, tt.want)
			}
		})
	}
}

func TestCentsString(t *testing.T) {
	tests := []struct {
		c    Cents
		want string
	}{
		{0, "0,00 €"},
		{5, "0,05 €"},
		{1234, "12,34 €"},
		{100000, "1000,00 €"},
		{-150, "-1,50 €"},
	}

	for _, tt := range tests {
		if got := tt.c.String(); got != tt.want {
			t.Errorf("Cents(%d).String() = %q, attendu %q", int64(tt.c), got, tt.want)
		}
	}
}
