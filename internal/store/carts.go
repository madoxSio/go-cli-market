package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/madoxSio/go-cli-market/internal/api"
	"github.com/madoxSio/go-cli-market/internal/market"
)

// querier est ce qu'ont en commun *sql.DB (hors transaction) et *sql.Tx (dans une transaction) :
// les fonctions de lecture du panier marchent ainsi dans les deux cas.
type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// GetCart renvoie le panier en cours du client, ou un panier vide s'il n'en a pas.
func (m *Market) GetCart(ctx context.Context, userID int64) (market.Cart, error) {
	var id int64
	var ref string
	err := m.db.QueryRowContext(ctx,
		`SELECT id, ref FROM carts WHERE user_id = $1 AND status = 'open'`, userID,
	).Scan(&id, &ref)
	if errors.Is(err, sql.ErrNoRows) {
		return emptyCart(), nil
	}
	if err != nil {
		return market.Cart{}, err
	}
	return loadCart(ctx, m.db, id, ref)
}

// AddItems ajoute des produits au panier en cours (créé s'il n'existe pas).
// Si un produit y est déjà, les quantités s'additionnent.
func (m *Market) AddItems(ctx context.Context, userID int64, items []api.ItemRequest) (market.Cart, error) {
	return m.changeCart(ctx, userID, true, func(tx *sql.Tx, cartID int64) error {
		for _, it := range items {
			productID, err := productIDByRef(ctx, tx, it.ProductRef)
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `
				INSERT INTO cart_items (cart_id, product_id, quantity) VALUES ($1, $2, $3)
				ON CONFLICT (cart_id, product_id) DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity`,
				cartID, productID, it.Quantity)
			if err != nil {
				return quantityError(err)
			}
		}
		return nil
	})
}

// SetQuantities remplace les quantités de produits du panier. Une quantité de 0 retire le produit.
func (m *Market) SetQuantities(ctx context.Context, userID int64, items []api.ItemRequest) (market.Cart, error) {
	return m.changeCart(ctx, userID, true, func(tx *sql.Tx, cartID int64) error {
		for _, it := range items {
			productID, err := productIDByRef(ctx, tx, it.ProductRef)
			if err != nil {
				return err
			}
			if it.Quantity == 0 {
				_, err = tx.ExecContext(ctx,
					`DELETE FROM cart_items WHERE cart_id = $1 AND product_id = $2`, cartID, productID)
			} else {
				_, err = tx.ExecContext(ctx, `
					INSERT INTO cart_items (cart_id, product_id, quantity) VALUES ($1, $2, $3)
					ON CONFLICT (cart_id, product_id) DO UPDATE SET quantity = EXCLUDED.quantity`,
					cartID, productID, it.Quantity)
			}
			if err != nil {
				return quantityError(err)
			}
		}
		return nil
	})
}

// RemoveItem retire un produit du panier en cours. market.ErrNotFound s'il n'y est pas.
func (m *Market) RemoveItem(ctx context.Context, userID int64, productRef string) (market.Cart, error) {
	return m.changeCart(ctx, userID, false, func(tx *sql.Tx, cartID int64) error {
		res, err := tx.ExecContext(ctx, `
			DELETE FROM cart_items
			WHERE cart_id = $1 AND product_id = (SELECT id FROM products WHERE ref = $2)`,
			cartID, productRef)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("produit %s absent du panier : %w", productRef, market.ErrNotFound)
		}
		return nil
	})
}

// changeCart exécute change dans une transaction, sur le panier en cours du client
// (créé si create vaut true), puis renvoie le panier à jour.
// Sans panier en cours et avec create à false, renvoie market.ErrNotFound.
func (m *Market) changeCart(ctx context.Context, userID int64, create bool, change func(tx *sql.Tx, cartID int64) error) (market.Cart, error) {
	var cart market.Cart
	err := m.withTx(ctx, func(tx *sql.Tx) error {
		cartID, ref, err := openCart(ctx, tx, userID, create)
		if err != nil {
			return err
		}
		if cartID == 0 {
			return fmt.Errorf("aucun panier en cours : %w", market.ErrNotFound)
		}
		if err := change(tx, cartID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE carts SET updated_at = NOW() WHERE id = $1`, cartID); err != nil {
			return err
		}
		cart, err = loadCart(ctx, tx, cartID, ref)
		return err
	})
	return cart, err
}

// withTx ouvre une transaction, exécute fn, puis valide (COMMIT) si fn réussit
// ou annule tout (ROLLBACK) si fn renvoie une erreur.
func (m *Market) withTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // sans effet si Commit a déjà réussi

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// openCart renvoie l'id et la ref du panier en cours du client, en le verrouillant jusqu'à
// la fin de la transaction (FOR UPDATE). Sans panier : en crée un si create, sinon renvoie 0.
func openCart(ctx context.Context, tx *sql.Tx, userID int64, create bool) (int64, string, error) {
	for attempt := 1; ; attempt++ {
		var id int64
		var ref string
		err := tx.QueryRowContext(ctx,
			`SELECT id, ref FROM carts WHERE user_id = $1 AND status = 'open' FOR UPDATE`, userID,
		).Scan(&id, &ref)
		if err == nil {
			return id, ref, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, "", err
		}
		if !create {
			return 0, "", nil
		}
		if attempt > maxRefAttempts {
			return 0, "", errors.New("impossible de créer le panier")
		}
		// ON CONFLICT DO NOTHING : si la ref est déjà prise, ou si une autre requête vient de
		// créer le panier du client, l'insertion est ignorée et la boucle relit le panier.
		_, err = tx.ExecContext(ctx,
			`INSERT INTO carts (ref, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			newRef(market.PrefixCart), userID)
		if isForeignKeyViolation(err) {
			return 0, "", fmt.Errorf("utilisateur %d : %w", userID, market.ErrNotFound)
		}
		if err != nil {
			return 0, "", err
		}
	}
}

// loadCart lit les lignes d'un panier avec les prix actuels des produits.
func loadCart(ctx context.Context, q querier, cartID int64, ref string) (market.Cart, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT p.ref, p.name, p.price_ht, p.vat_rate, ci.quantity
		FROM cart_items ci
		JOIN products p ON p.id = ci.product_id
		WHERE ci.cart_id = $1
		ORDER BY ci.added_at, p.id`, cartID)
	if err != nil {
		return market.Cart{}, err
	}
	defer rows.Close()

	cart := emptyCart()
	cart.Ref = ref
	for rows.Next() {
		var p market.Product
		var quantity int
		if err := rows.Scan(&p.Ref, &p.Name, &p.PriceHT, &p.VATRate, &quantity); err != nil {
			return market.Cart{}, err
		}
		cart.Items = append(cart.Items, market.NewItem(p, quantity))
	}
	cart.TotalTTC = market.Total(cart.Items)
	return cart, rows.Err()
}

// emptyCart renvoie un panier sans ligne (Items vaut [] et non null en JSON).
func emptyCart() market.Cart {
	return market.Cart{Items: []market.Item{}}
}

// productIDByRef renvoie l'id interne d'un produit, ou market.ErrNotFound.
func productIDByRef(ctx context.Context, q querier, ref string) (int64, error) {
	var id int64
	err := q.QueryRowContext(ctx, `SELECT id FROM products WHERE ref = $1`, ref).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("produit %s : %w", ref, market.ErrNotFound)
	}
	return id, err
}

// quantityError traduit le refus de la contrainte CHECK sur la quantité (code 23514)
// en erreur métier : c'est le cas d'un ajout qui ferait dépasser 99 exemplaires.
func quantityError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23514" {
		return fmt.Errorf("la quantité d'un produit ne peut pas dépasser %d : %w", market.MaxQuantity, market.ErrInvalidInput)
	}
	return err
}
