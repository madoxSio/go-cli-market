package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/madoxSio/go-cli-market/internal/market"
)

// ttcSQL calcule le prix TTC en SQL, avec exactement la même formule que market.PriceWithVAT,
// pour pouvoir filtrer sur le TTC dans la base.
const ttcSQL = `((price_ht * (10000 + vat_rate) + 5000) / 10000)`

const productSelect = `
	SELECT id, ref, name, description, category, price_ht, vat_rate, created_at
	FROM products`

// CreateProduct enregistre un produit déjà validé (market.ProductInput.Validate)
// et lui attribue un identifiant métier PDT- unique.
func (m *Market) CreateProduct(ctx context.Context, p market.Product) (market.Product, error) {
	for attempt := 1; ; attempt++ {
		p.Ref = newRef(market.PrefixProduct)
		err := m.db.QueryRowContext(ctx, `
			INSERT INTO products (ref, name, description, category, price_ht, vat_rate)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, created_at`,
			p.Ref, p.Name, p.Description, p.Category, p.PriceHT, p.VATRate,
		).Scan(&p.ID, &p.CreatedAt)
		if err == nil {
			p.PriceTTC = market.PriceWithVAT(p.PriceHT, p.VATRate)
			return p, nil
		}
		if !isUniqueViolation(err, "products_ref_key") || attempt == maxRefAttempts {
			return market.Product{}, fmt.Errorf("création du produit : %w", err)
		}
		// Identifiant déjà pris : on en tire un autre.
	}
}

// GetProduct renvoie le produit d'identifiant métier ref, ou market.ErrNotFound.
func (m *Market) GetProduct(ctx context.Context, ref string) (market.Product, error) {
	row := m.db.QueryRowContext(ctx, productSelect+` WHERE ref = $1`, ref)
	p, err := scanProduct(row)
	if errors.Is(err, sql.ErrNoRows) {
		return market.Product{}, fmt.Errorf("produit %s : %w", ref, market.ErrNotFound)
	}
	return p, err
}

// SearchProducts renvoie les produits qui correspondent au filtre, triés par nom.
// Seuls les critères renseignés sont ajoutés à la requête ; toutes les valeurs passent
// par des paramètres ($1, $2…), jamais dans le texte SQL.
func (m *Market) SearchProducts(ctx context.Context, f market.ProductFilter) ([]market.Product, error) {
	var conds []string
	var args []any
	// add ajoute une condition dont le %d sera remplacé par le numéro du paramètre.
	add := func(cond string, value any) {
		args = append(args, value)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}

	if q := strings.TrimSpace(f.Query); q != "" {
		add(`(name ILIKE $%[1]d OR description ILIKE $%[1]d)`, "%"+escapeLike(q)+"%")
	}
	if c := strings.TrimSpace(f.Category); c != "" {
		add(`category = $%d`, strings.ToLower(c))
	}
	if f.MinHT != nil {
		add(`price_ht >= $%d`, *f.MinHT)
	}
	if f.MaxHT != nil {
		add(`price_ht <= $%d`, *f.MaxHT)
	}
	if f.MinTTC != nil {
		add(ttcSQL+` >= $%d`, *f.MinTTC)
	}
	if f.MaxTTC != nil {
		add(ttcSQL+` <= $%d`, *f.MaxTTC)
	}

	query := productSelect
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += " ORDER BY name, id"

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("recherche de produits : %w", err)
	}
	defer rows.Close()

	products := []market.Product{} // liste vide (et non nil) : le JSON sera [] et pas null
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

// scanProduct lit une ligne de productSelect et calcule le prix TTC.
func scanProduct(row scanner) (market.Product, error) {
	var p market.Product
	err := row.Scan(&p.ID, &p.Ref, &p.Name, &p.Description, &p.Category, &p.PriceHT, &p.VATRate, &p.CreatedAt)
	if err != nil {
		return market.Product{}, err
	}
	p.PriceTTC = market.PriceWithVAT(p.PriceHT, p.VATRate)
	return p, nil
}

// escapeLike neutralise les caractères spéciaux de LIKE : sans ça, chercher « 100% »
// trouverait tout ce qui commence par « 100 ».
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
