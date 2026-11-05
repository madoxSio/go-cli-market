package market

import "fmt"

// MaxQuantity est la quantité maximale d'un même produit dans un panier.
const MaxQuantity = 99

// Item est une ligne de panier ou de commande : un produit, son prix au moment
// de l'ajout et une quantité. Dans une commande, ces prix sont figés.
type Item struct {
	ProductRef   string      `json:"product_ref"`
	Name         string      `json:"name"`
	UnitPriceHT  Cents       `json:"unit_price_ht"`
	VATRate      BasisPoints `json:"vat_rate"`
	UnitPriceTTC Cents       `json:"unit_price_ttc"`
	Quantity     int         `json:"quantity"`
	LineTTC      Cents       `json:"line_total_ttc"`
}

// NewItem crée la ligne d'un produit avec ses prix calculés.
func NewItem(p Product, quantity int) Item {
	unitTTC := PriceWithVAT(p.PriceHT, p.VATRate)
	return Item{
		ProductRef:   p.Ref,
		Name:         p.Name,
		UnitPriceHT:  p.PriceHT,
		VATRate:      p.VATRate,
		UnitPriceTTC: unitTTC,
		Quantity:     quantity,
		LineTTC:      unitTTC * Cents(quantity),
	}
}

// Cart est le panier en cours d'un client. La livraison est toujours gratuite.
type Cart struct {
	Ref      string `json:"ref"`
	Items    []Item `json:"items"`
	Shipping Cents  `json:"shipping"`
	TotalTTC Cents  `json:"total_ttc"`
}

// Total additionne les lignes. Le prix TTC est arrondi par unité, puis multiplié par la quantité.
func Total(items []Item) Cents {
	var total Cents
	for _, it := range items {
		total += it.LineTTC
	}
	return total
}

// ValidateQuantity vérifie une quantité saisie par le client (de 1 à MaxQuantity).
func ValidateQuantity(q int) error {
	if q < 1 || q > MaxQuantity {
		return fmt.Errorf("la quantité doit être entre 1 et %d : %w", MaxQuantity, ErrInvalidInput)
	}
	return nil
}
