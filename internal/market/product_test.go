package market

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestProductInputValidate(t *testing.T) {
	reduced := VATReduced
	zero := BasisPoints(0)
	tooHigh := BasisPoints(10001)

	tests := []struct {
		name    string
		in      ProductInput
		wantErr bool
		check   func(t *testing.T, p Product)
	}{
		{
			name: "TVA par défaut à 20 %",
			in:   ProductInput{Name: "Clavier", Category: "Informatique", PriceHT: 5000},
			check: func(t *testing.T, p Product) {
				if p.VATRate != VATStandard || p.PriceTTC != 6000 {
					t.Errorf("TVA = %d, TTC = %d ; attendu 2000 et 6000", p.VATRate, p.PriceTTC)
				}
			},
		},
		{
			name: "TVA précisée à 5,5 %",
			in:   ProductInput{Name: "Livre", Category: "livres", PriceHT: 1000, VATRate: &reduced},
			check: func(t *testing.T, p Product) {
				if p.VATRate != VATReduced || p.PriceTTC != 1055 {
					t.Errorf("TVA = %d, TTC = %d ; attendu 550 et 1055", p.VATRate, p.PriceTTC)
				}
			},
		},
		{
			name: "TVA à 0 % acceptée (différente de « non précisée »)",
			in:   ProductInput{Name: "Don", Category: "divers", PriceHT: 1000, VATRate: &zero},
			check: func(t *testing.T, p Product) {
				if p.VATRate != 0 || p.PriceTTC != 1000 {
					t.Errorf("TVA = %d, TTC = %d ; attendu 0 et 1000", p.VATRate, p.PriceTTC)
				}
			},
		},
		{
			name: "espaces retirés, catégorie en minuscules",
			in:   ProductInput{Name: "  Souris  ", Description: " sans fil ", Category: " Informatique ", PriceHT: 2500},
			check: func(t *testing.T, p Product) {
				if p.Name != "Souris" || p.Description != "sans fil" || p.Category != "informatique" {
					t.Errorf("produit nettoyé = %+v", p)
				}
			},
		},
		{name: "nom vide", in: ProductInput{Name: "   ", Category: "x", PriceHT: 100}, wantErr: true},
		{name: "nom trop long", in: ProductInput{Name: strings.Repeat("é", 201), Category: "x", PriceHT: 100}, wantErr: true},
		{name: "catégorie vide", in: ProductInput{Name: "A", PriceHT: 100}, wantErr: true},
		{name: "prix nul", in: ProductInput{Name: "A", Category: "x", PriceHT: 0}, wantErr: true},
		{name: "prix négatif", in: ProductInput{Name: "A", Category: "x", PriceHT: -5}, wantErr: true},
		{name: "TVA trop haute", in: ProductInput{Name: "A", Category: "x", PriceHT: 100, VATRate: &tooHigh}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := tt.in.Validate()
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf("erreur = %v, attendu ErrInvalidInput", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("erreur inattendue : %v", err)
			}
			tt.check(t, p)
		})
	}
}

func TestProductJSON(t *testing.T) {
	p := Product{ID: 7, Ref: "PDT-7D2K8N", Name: "Clavier", PriceHT: 5000, VATRate: 2000, PriceTTC: 6000}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{`"ref":"PDT-7D2K8N"`, `"price_ht":5000`, `"price_ttc":6000`, `"vat_rate":2000`} {
		if !strings.Contains(got, want) {
			t.Errorf("JSON %s ne contient pas %s", got, want)
		}
	}
	if strings.Contains(got, `"ID"`) || strings.Contains(got, `"id"`) {
		t.Errorf("l'identifiant interne ne doit pas apparaître dans le JSON : %s", got)
	}
}
