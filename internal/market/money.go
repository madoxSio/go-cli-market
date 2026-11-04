package market

import "fmt"

// Cents est un montant en centimes d'euro : 1234 vaut 12,34 €.
// On n'utilise jamais de float64 pour l'argent, car 0.1 + 0.2 n'y vaut pas 0.3.
type Cents int64

// BasisPoints est un taux en points de base : 2000 vaut 20 %, 550 vaut 5,5 %.
type BasisPoints int64

// Taux de TVA français.
const (
	VATStandard     BasisPoints = 2000 // 20 %
	VATIntermediate BasisPoints = 1000 // 10 %
	VATReduced      BasisPoints = 550  // 5,5 %
	VATSuper        BasisPoints = 210  // 2,1 %
)

// PriceWithVAT calcule le prix TTC à partir du prix HT, arrondi au centime le plus proche
// (un demi-centime est arrondi vers le haut). Le prix HT et le taux doivent être positifs.
func PriceWithVAT(ht Cents, rate BasisPoints) Cents {
	// TTC = HT × (10000 + taux) / 10000. Ajouter 5000 avant la division entière arrondit au plus proche.
	return Cents((int64(ht)*(10000+int64(rate)) + 5000) / 10000)
}

// String affiche le montant au format français, par exemple « 12,34 € ».
func (c Cents) String() string {
	sign := ""
	if c < 0 {
		sign = "-"
		c = -c
	}
	return fmt.Sprintf("%s%d,%02d €", sign, c/100, c%100)
}
