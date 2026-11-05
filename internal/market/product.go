package market

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Product est un article du catalogue. Le prix TTC n'est pas stocké en base :
// il est calculé à partir du prix HT et du taux de TVA.
type Product struct {
	ID          int64       `json:"-"`
	Ref         string      `json:"ref"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Category    string      `json:"category"`
	PriceHT     Cents       `json:"price_ht"`
	VATRate     BasisPoints `json:"vat_rate"`
	PriceTTC    Cents       `json:"price_ttc"`
	CreatedAt   time.Time   `json:"created_at"`
}

// ProductInput est ce que l'administrateur envoie pour créer un produit.
// VATRate est un pointeur : nil veut dire « non précisé », et vaut alors 20 %.
type ProductInput struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Category    string       `json:"category"`
	PriceHT     Cents        `json:"price_ht"`
	VATRate     *BasisPoints `json:"vat_rate"`
}

// Validate nettoie la saisie, vérifie les règles et renvoie le produit prêt à enregistrer
// (sans ID ni Ref, attribués par le store).
func (in ProductInput) Validate() (Product, error) {
	p := Product{
		Name:        strings.TrimSpace(in.Name),
		Description: strings.TrimSpace(in.Description),
		Category:    strings.ToLower(strings.TrimSpace(in.Category)),
		PriceHT:     in.PriceHT,
		VATRate:     VATStandard,
	}
	if in.VATRate != nil {
		p.VATRate = *in.VATRate
	}

	switch {
	case p.Name == "" || utf8.RuneCountInString(p.Name) > 200:
		return Product{}, fmt.Errorf("le nom est obligatoire (200 caractères maximum) : %w", ErrInvalidInput)
	case utf8.RuneCountInString(p.Description) > 2000:
		return Product{}, fmt.Errorf("la description dépasse 2000 caractères : %w", ErrInvalidInput)
	case p.Category == "" || utf8.RuneCountInString(p.Category) > 100:
		return Product{}, fmt.Errorf("la catégorie est obligatoire (100 caractères maximum) : %w", ErrInvalidInput)
	case p.PriceHT <= 0:
		return Product{}, fmt.Errorf("le prix HT doit être strictement positif : %w", ErrInvalidInput)
	case p.VATRate < 0 || p.VATRate > 10000:
		return Product{}, fmt.Errorf("le taux de TVA doit être entre 0 et 10000 points de base : %w", ErrInvalidInput)
	}

	p.PriceTTC = PriceWithVAT(p.PriceHT, p.VATRate)
	return p, nil
}

// ProductFilter décrit une recherche dans le catalogue. Un champ vide ou nil n'est pas filtré.
// Les bornes de prix sont des pointeurs pour distinguer « pas de borne » de « borne à 0 ».
type ProductFilter struct {
	Query    string // cherché dans le nom et la description, sans tenir compte de la casse
	Category string
	MinHT    *Cents
	MaxHT    *Cents
	MinTTC   *Cents
	MaxTTC   *Cents
}
