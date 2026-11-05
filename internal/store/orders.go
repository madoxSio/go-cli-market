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

const orderSelect = `
	SELECT o.id, o.ref, o.user_id, u.email, o.status, o.cancel_reason,
	       o.total_ttc, o.card_last4, o.created_at, o.updated_at
	FROM orders o
	JOIN users u ON u.id = o.user_id`

// Checkout paie le panier en cours du client : dans une seule transaction, la commande est
// créée au statut « paid » avec une copie des lignes, puis le panier passe à « checked_out ».
// cardLast4 vient de market.ValidateCard. Panier absent ou vide : market.ErrConflict.
func (m *Market) Checkout(ctx context.Context, userID int64, cardLast4 string) (market.Order, error) {
	var orderID int64
	err := m.withTx(ctx, func(tx *sql.Tx) error {
		cartID, cartRef, err := openCart(ctx, tx, userID, false)
		if err != nil {
			return err
		}
		if cartID == 0 {
			return fmt.Errorf("le panier est vide : %w", market.ErrConflict)
		}
		cart, err := loadCart(ctx, tx, cartID, cartRef)
		if err != nil {
			return err
		}
		if len(cart.Items) == 0 {
			return fmt.Errorf("le panier est vide : %w", market.ErrConflict)
		}

		orderID, err = insertOrder(ctx, tx, userID, sql.NullInt64{Int64: cartID, Valid: true},
			market.StatusPaid, cart.Items, cardLast4)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			`UPDATE carts SET status = 'checked_out', updated_at = NOW() WHERE id = $1`, cartID)
		return err
	})
	if err != nil {
		return market.Order{}, err
	}
	return m.orderByID(ctx, orderID)
}

// CreateOrder crée, pour le compte d'un administrateur, une commande « pending » pour un client,
// avec les prix actuels des produits. Un même produit cité deux fois voit ses quantités additionnées.
func (m *Market) CreateOrder(ctx context.Context, userID int64, requested []api.ItemRequest) (market.Order, error) {
	var orderID int64
	err := m.withTx(ctx, func(tx *sql.Tx) error {
		var items []market.Item
		position := map[string]int{} // ref du produit → indice dans items
		for _, req := range requested {
			if i, seen := position[req.ProductRef]; seen {
				items[i] = market.NewItem(productOf(items[i]), items[i].Quantity+req.Quantity)
				continue
			}
			row := tx.QueryRowContext(ctx, productSelect+` WHERE ref = $1`, req.ProductRef)
			p, err := scanProduct(row)
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("produit %s : %w", req.ProductRef, market.ErrNotFound)
			}
			if err != nil {
				return err
			}
			position[req.ProductRef] = len(items)
			items = append(items, market.NewItem(p, req.Quantity))
		}

		var err error
		orderID, err = insertOrder(ctx, tx, userID, sql.NullInt64{}, market.StatusPending, items, "")
		return err
	})
	if err != nil {
		return market.Order{}, err
	}
	return m.orderByID(ctx, orderID)
}

// productOf reconstitue le produit d'une ligne, pour recalculer la ligne avec une autre quantité.
func productOf(it market.Item) market.Product {
	return market.Product{Ref: it.ProductRef, Name: it.Name, PriceHT: it.UnitPriceHT, VATRate: it.VATRate}
}

// insertOrder enregistre une commande et ses lignes (prix copiés) et renvoie son id.
func insertOrder(ctx context.Context, tx *sql.Tx, userID int64, cartID sql.NullInt64,
	status market.OrderStatus, items []market.Item, cardLast4 string) (int64, error) {

	var orderID int64
	for attempt := 1; ; attempt++ {
		// ON CONFLICT (ref) DO NOTHING : si la ref tirée existe déjà, aucune ligne n'est renvoyée
		// (sql.ErrNoRows) et on recommence. Une vraie erreur annulerait toute la transaction.
		err := tx.QueryRowContext(ctx, `
			INSERT INTO orders (ref, user_id, cart_id, status, total_ttc, card_last4)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (ref) DO NOTHING
			RETURNING id`,
			newRef(market.PrefixOrder), userID, cartID, status, market.Total(items), cardLast4,
		).Scan(&orderID)
		if err == nil {
			break
		}
		if isForeignKeyViolation(err) {
			return 0, fmt.Errorf("utilisateur %d : %w", userID, market.ErrNotFound)
		}
		if !errors.Is(err, sql.ErrNoRows) || attempt == maxRefAttempts {
			return 0, fmt.Errorf("création de la commande : %w", err)
		}
	}

	for i, it := range items {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO order_items (order_id, position, product_id, product_ref, name,
			                         unit_price_ht, vat_rate, unit_price_ttc, quantity)
			VALUES ($1, $2, (SELECT id FROM products WHERE ref = $3), $3, $4, $5, $6, $7, $8)`,
			orderID, i, it.ProductRef, it.Name, it.UnitPriceHT, it.VATRate, it.UnitPriceTTC, it.Quantity)
		if err != nil {
			return 0, fmt.Errorf("ligne de commande %s : %w", it.ProductRef, err)
		}
	}
	return orderID, nil
}

// GetOrder renvoie la commande d'identifiant métier ref, ou market.ErrNotFound.
func (m *Market) GetOrder(ctx context.Context, ref string) (market.Order, error) {
	orders, err := m.queryOrders(ctx, orderSelect+` WHERE o.ref = $1`, ref)
	if err != nil {
		return market.Order{}, err
	}
	if len(orders) == 0 {
		return market.Order{}, fmt.Errorf("commande %s : %w", ref, market.ErrNotFound)
	}
	return orders[0], nil
}

// ListUserOrders renvoie les commandes d'un client, de la plus récente à la plus ancienne.
func (m *Market) ListUserOrders(ctx context.Context, userID int64) ([]market.Order, error) {
	return m.queryOrders(ctx, orderSelect+` WHERE o.user_id = $1 ORDER BY o.created_at DESC, o.id DESC`, userID)
}

// ListOrders renvoie les commandes de tous les clients, éventuellement filtrées par statut ("" = toutes).
func (m *Market) ListOrders(ctx context.Context, status market.OrderStatus) ([]market.Order, error) {
	if status == "" {
		return m.queryOrders(ctx, orderSelect+` ORDER BY o.created_at DESC, o.id DESC`)
	}
	return m.queryOrders(ctx, orderSelect+` WHERE o.status = $1 ORDER BY o.created_at DESC, o.id DESC`, status)
}

// UpdateOrderStatus change le statut d'une commande si la transition est permise
// (market.ValidateTransition). La raison n'est enregistrée que pour une annulation.
func (m *Market) UpdateOrderStatus(ctx context.Context, ref string, to market.OrderStatus, reason string) (market.Order, error) {
	var orderID int64
	err := m.withTx(ctx, func(tx *sql.Tx) error {
		var from market.OrderStatus
		err := tx.QueryRowContext(ctx,
			`SELECT id, status FROM orders WHERE ref = $1 FOR UPDATE`, ref,
		).Scan(&orderID, &from)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("commande %s : %w", ref, market.ErrNotFound)
		}
		if err != nil {
			return err
		}
		if err := market.ValidateTransition(from, to, reason); err != nil {
			return err
		}
		cancelReason := ""
		if to == market.StatusCancelled {
			cancelReason = reason
		}
		_, err = tx.ExecContext(ctx,
			`UPDATE orders SET status = $1, cancel_reason = $2, updated_at = NOW() WHERE id = $3`,
			to, cancelReason, orderID)
		return err
	})
	if err != nil {
		return market.Order{}, err
	}
	return m.orderByID(ctx, orderID)
}

func (m *Market) orderByID(ctx context.Context, id int64) (market.Order, error) {
	orders, err := m.queryOrders(ctx, orderSelect+` WHERE o.id = $1`, id)
	if err != nil {
		return market.Order{}, err
	}
	if len(orders) == 0 {
		return market.Order{}, fmt.Errorf("commande %d : %w", id, market.ErrNotFound)
	}
	return orders[0], nil
}

// queryOrders exécute une requête basée sur orderSelect, puis charge les lignes de toutes
// les commandes trouvées en une seule requête supplémentaire.
func (m *Market) queryOrders(ctx context.Context, query string, args ...any) ([]market.Order, error) {
	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("lecture des commandes : %w", err)
	}
	defer rows.Close()

	orders := []market.Order{}
	index := map[int64]int{} // id de la commande → position dans orders
	var ids []int64
	for rows.Next() {
		var o market.Order
		err := rows.Scan(&o.ID, &o.Ref, &o.UserID, &o.UserEmail, &o.Status, &o.CancelReason,
			&o.TotalTTC, &o.CardLast4, &o.CreatedAt, &o.UpdatedAt)
		if err != nil {
			return nil, err
		}
		o.Items = []market.Item{}
		index[o.ID] = len(orders)
		ids = append(ids, o.ID)
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return orders, nil
	}

	itemRows, err := m.db.QueryContext(ctx, `
		SELECT order_id, product_ref, name, unit_price_ht, vat_rate, unit_price_ttc, quantity
		FROM order_items
		WHERE order_id = ANY($1)
		ORDER BY order_id, position`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("lecture des lignes de commande : %w", err)
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var orderID int64
		var it market.Item
		err := itemRows.Scan(&orderID, &it.ProductRef, &it.Name, &it.UnitPriceHT, &it.VATRate, &it.UnitPriceTTC, &it.Quantity)
		if err != nil {
			return nil, err
		}
		it.LineTTC = it.UnitPriceTTC * market.Cents(it.Quantity)
		o := &orders[index[orderID]]
		o.Items = append(o.Items, it)
	}
	return orders, itemRows.Err()
}
