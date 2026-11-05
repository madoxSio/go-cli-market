package store

import (
	"context"
	"errors"
	"testing"

	"github.com/madoxSio/go-cli-market/internal/api"
	"github.com/madoxSio/go-cli-market/internal/market"
)

func TestCheckout(t *testing.T) {
	db := openTestDB(t)
	m := NewMarket(db)
	ctx := context.Background()
	category := testCategory(t, m)
	user := testUser(t, db)
	keyboard := mustCreateProduct(t, m, market.ProductInput{Name: "Clavier", Category: category, PriceHT: 5000}) // 60,00 €
	mouse := mustCreateProduct(t, m, market.ProductInput{Name: "Souris", Category: category, PriceHT: 2000})     // 24,00 €

	// Panier vide : impossible de payer.
	if _, err := m.Checkout(ctx, user, "4242"); !errors.Is(err, market.ErrConflict) {
		t.Fatalf("paiement sans panier : %v, attendu ErrConflict", err)
	}

	cart, err := m.AddItems(ctx, user, []api.ItemRequest{item(keyboard.Ref, 1), item(mouse.Ref, 2)})
	if err != nil {
		t.Fatal(err)
	}

	order, err := m.Checkout(ctx, user, "4242")
	if err != nil {
		t.Fatal(err)
	}
	if !market.IsBusinessID(order.Ref, market.PrefixOrder) || order.Status != market.StatusPaid ||
		order.UserID != user || order.CardLast4 != "4242" || order.TotalTTC != 6000+4800 || len(order.Items) != 2 {
		t.Fatalf("commande = %+v", order)
	}
	if order.Items[1].Name != "Souris" || order.Items[1].Quantity != 2 || order.Items[1].LineTTC != 4800 {
		t.Errorf("ligne 2 = %+v", order.Items[1])
	}

	// Le panier payé est figé : le panier en cours est vide, et un nouvel ajout crée un nouveau panier.
	if again, _ := m.GetCart(ctx, user); again.Ref != "" || len(again.Items) != 0 {
		t.Errorf("après paiement, panier = %+v", again)
	}
	next, err := m.AddItems(ctx, user, []api.ItemRequest{item(mouse.Ref, 1)})
	if err != nil || next.Ref == cart.Ref {
		t.Errorf("nouveau panier = %+v (ancien %s), err %v", next, cart.Ref, err)
	}

	// Les prix de la commande sont figés : changer le prix du produit ne la modifie pas.
	if _, err := db.Exec(`UPDATE products SET price_ht = 9999 WHERE ref = $1`, keyboard.Ref); err != nil {
		t.Fatal(err)
	}
	reread, err := m.GetOrder(ctx, order.Ref)
	if err != nil || reread.TotalTTC != order.TotalTTC || reread.Items[0].UnitPriceHT != 5000 {
		t.Errorf("commande relue après changement de prix = %+v, err %v", reread, err)
	}
}

func TestCreateOrderAndStatus(t *testing.T) {
	db := openTestDB(t)
	m := NewMarket(db)
	ctx := context.Background()
	category := testCategory(t, m)
	user := testUser(t, db)
	p := mustCreateProduct(t, m, market.ProductInput{Name: "Livre", Category: category, PriceHT: 1000}) // 12,00 €

	// Commande admin : statut pending, quantités d'un même produit additionnées, pas de carte.
	order, err := m.CreateOrder(ctx, user, []api.ItemRequest{item(p.Ref, 1), item(p.Ref, 2)})
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != market.StatusPending || len(order.Items) != 1 || order.Items[0].Quantity != 3 ||
		order.TotalTTC != 3600 || order.CardLast4 != "" {
		t.Fatalf("commande admin = %+v", order)
	}

	if _, err := m.CreateOrder(ctx, user, []api.ItemRequest{item("PDT-ZZZZZZ", 1)}); !errors.Is(err, market.ErrNotFound) {
		t.Errorf("produit inconnu : %v, attendu ErrNotFound", err)
	}
	if _, err := m.CreateOrder(ctx, -1, []api.ItemRequest{item(p.Ref, 1)}); !errors.Is(err, market.ErrNotFound) {
		t.Errorf("client inconnu : %v, attendu ErrNotFound", err)
	}

	// Changements de statut.
	order, err = m.UpdateOrderStatus(ctx, order.Ref, market.StatusPaid, "")
	if err != nil || order.Status != market.StatusPaid {
		t.Fatalf("paiement : %+v, err %v", order, err)
	}
	if _, err := m.UpdateOrderStatus(ctx, order.Ref, market.StatusDelivered, ""); !errors.Is(err, market.ErrConflict) {
		t.Errorf("paid → delivered : %v, attendu ErrConflict", err)
	}
	if _, err := m.UpdateOrderStatus(ctx, order.Ref, market.StatusCancelled, ""); !errors.Is(err, market.ErrInvalidInput) {
		t.Errorf("annulation sans raison : %v, attendu ErrInvalidInput", err)
	}
	order, err = m.UpdateOrderStatus(ctx, order.Ref, market.StatusCancelled, "Rupture de stock")
	if err != nil || order.Status != market.StatusCancelled || order.CancelReason != "Rupture de stock" {
		t.Fatalf("annulation : %+v, err %v", order, err)
	}
	if _, err := m.UpdateOrderStatus(ctx, "CMD-ZZZZZZ", market.StatusPaid, ""); !errors.Is(err, market.ErrNotFound) {
		t.Errorf("commande inconnue : %v, attendu ErrNotFound", err)
	}
}

func TestListOrders(t *testing.T) {
	db := openTestDB(t)
	m := NewMarket(db)
	ctx := context.Background()
	category := testCategory(t, m)
	alice, bob := testUser(t, db), testUser(t, db)
	p := mustCreateProduct(t, m, market.ProductInput{Name: "A", Category: category, PriceHT: 100})

	first, _ := m.CreateOrder(ctx, alice, []api.ItemRequest{item(p.Ref, 1)})
	second, _ := m.CreateOrder(ctx, alice, []api.ItemRequest{item(p.Ref, 2)})
	other, _ := m.CreateOrder(ctx, bob, []api.ItemRequest{item(p.Ref, 3)})
	if _, err := m.UpdateOrderStatus(ctx, other.Ref, market.StatusPaid, ""); err != nil {
		t.Fatal(err)
	}

	// Les commandes d'Alice seulement, la plus récente d'abord, avec leurs lignes.
	mine, err := m.ListUserOrders(ctx, alice)
	if err != nil || len(mine) != 2 || mine[0].Ref != second.Ref || mine[1].Ref != first.Ref || len(mine[0].Items) != 1 {
		t.Fatalf("commandes d'Alice = %+v, err %v", mine, err)
	}

	// Un client sans commande : liste vide, pas nil.
	none, err := m.ListUserOrders(ctx, testUser(t, db))
	if err != nil || none == nil || len(none) != 0 {
		t.Errorf("sans commande = %v, err %v", none, err)
	}

	// Vue admin filtrée par statut : la commande payée de Bob en fait partie, pas celles d'Alice.
	paid, err := m.ListOrders(ctx, market.StatusPaid)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, o := range paid {
		if o.Status != market.StatusPaid {
			t.Errorf("commande %s au statut %s dans le filtre paid", o.Ref, o.Status)
		}
		if o.Ref == other.Ref {
			found = true
			if o.UserEmail == "" {
				t.Error("la vue admin doit contenir l'email du client")
			}
		}
		if o.Ref == first.Ref || o.Ref == second.Ref {
			t.Errorf("commande pending %s dans le filtre paid", o.Ref)
		}
	}
	if !found {
		t.Error("la commande payée de Bob est absente")
	}
}
