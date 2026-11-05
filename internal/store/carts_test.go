package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/madoxSio/go-cli-market/internal/api"
	"github.com/madoxSio/go-cli-market/internal/market"
)

// testUser crée un compte client directement en base et le supprime à la fin du test
// (ses paniers et commandes partent avec lui, grâce à ON DELETE CASCADE).
func testUser(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	var id int64
	email := fmt.Sprintf("store-%d@test.local", time.Now().UnixNano())
	err := db.QueryRow(`INSERT INTO users (email, password_hash, confirmed) VALUES ($1, 'x', TRUE) RETURNING id`, email).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM users WHERE id = $1`, id) })
	return id
}

func item(ref string, quantity int) api.ItemRequest {
	return api.ItemRequest{ProductRef: ref, Quantity: quantity}
}

func TestCartLifecycle(t *testing.T) {
	db := openTestDB(t)
	m := NewMarket(db)
	ctx := context.Background()
	user := testUser(t, db)
	category := testCategory(t, m)
	keyboard := mustCreateProduct(t, m, market.ProductInput{Name: "Clavier", Category: category, PriceHT: 5000}) // 60,00 € TTC
	mouse := mustCreateProduct(t, m, market.ProductInput{Name: "Souris", Category: category, PriceHT: 2000})     // 24,00 € TTC

	// Pas encore de panier : panier vide, sans ref.
	cart, err := m.GetCart(ctx, user)
	if err != nil || cart.Ref != "" || len(cart.Items) != 0 || cart.Items == nil {
		t.Fatalf("panier initial = %+v, err %v", cart, err)
	}

	// Ajouter plusieurs produits d'un coup crée le panier.
	cart, err = m.AddItems(ctx, user, []api.ItemRequest{item(keyboard.Ref, 1), item(mouse.Ref, 2)})
	if err != nil {
		t.Fatal(err)
	}
	if !market.IsBusinessID(cart.Ref, market.PrefixCart) || len(cart.Items) != 2 || cart.TotalTTC != 6000+2*2400 {
		t.Fatalf("après ajout = %+v", cart)
	}
	firstRef := cart.Ref

	// Ajouter un produit déjà présent additionne les quantités, dans le même panier.
	cart, err = m.AddItems(ctx, user, []api.ItemRequest{item(keyboard.Ref, 2)})
	if err != nil || cart.Ref != firstRef || cart.Items[0].Quantity != 3 {
		t.Fatalf("après second ajout = %+v, err %v", cart, err)
	}

	// Remplacer des quantités ; 0 retire le produit.
	cart, err = m.SetQuantities(ctx, user, []api.ItemRequest{item(keyboard.Ref, 1), item(mouse.Ref, 0)})
	if err != nil || len(cart.Items) != 1 || cart.Items[0].Quantity != 1 || cart.TotalTTC != 6000 {
		t.Fatalf("après modification = %+v, err %v", cart, err)
	}

	// Le panier relu est identique.
	again, err := m.GetCart(ctx, user)
	if err != nil || again.Ref != firstRef || again.TotalTTC != 6000 {
		t.Fatalf("panier relu = %+v, err %v", again, err)
	}

	// Retirer un produit absent : introuvable. Retirer le clavier : panier vide (mais toujours le même).
	if _, err := m.RemoveItem(ctx, user, mouse.Ref); !errors.Is(err, market.ErrNotFound) {
		t.Errorf("retrait d'un produit absent : %v, attendu ErrNotFound", err)
	}
	cart, err = m.RemoveItem(ctx, user, keyboard.Ref)
	if err != nil || len(cart.Items) != 0 || cart.Ref != firstRef || cart.TotalTTC != 0 {
		t.Fatalf("après retrait = %+v, err %v", cart, err)
	}
}

func TestCartErrors(t *testing.T) {
	db := openTestDB(t)
	m := NewMarket(db)
	ctx := context.Background()
	user := testUser(t, db)
	category := testCategory(t, m)
	p := mustCreateProduct(t, m, market.ProductInput{Name: "A", Category: category, PriceHT: 100})

	// Produit inconnu : rien n'est enregistré (la transaction est annulée).
	_, err := m.AddItems(ctx, user, []api.ItemRequest{item(p.Ref, 1), item("PDT-ZZZZZZ", 1)})
	if !errors.Is(err, market.ErrNotFound) {
		t.Fatalf("produit inconnu : %v, attendu ErrNotFound", err)
	}
	if cart, _ := m.GetCart(ctx, user); len(cart.Items) != 0 || cart.Ref != "" {
		t.Errorf("la transaction aurait dû tout annuler : %+v", cart)
	}

	// Dépasser 99 exemplaires en additionnant : refusé par la contrainte CHECK.
	if _, err := m.AddItems(ctx, user, []api.ItemRequest{item(p.Ref, 60)}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.AddItems(ctx, user, []api.ItemRequest{item(p.Ref, 40)}); !errors.Is(err, market.ErrInvalidInput) {
		t.Errorf("100 exemplaires : %v, attendu ErrInvalidInput", err)
	}

	// Retirer sans panier en cours : introuvable.
	other := testUser(t, db)
	if _, err := m.RemoveItem(ctx, other, p.Ref); !errors.Is(err, market.ErrNotFound) {
		t.Errorf("retrait sans panier : %v, attendu ErrNotFound", err)
	}
}

func TestCartUsesCurrentProductPrice(t *testing.T) {
	db := openTestDB(t)
	m := NewMarket(db)
	ctx := context.Background()
	user := testUser(t, db)
	category := testCategory(t, m)
	p := mustCreateProduct(t, m, market.ProductInput{Name: "A", Category: category, PriceHT: 1000})

	if _, err := m.AddItems(ctx, user, []api.ItemRequest{item(p.Ref, 1)}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE products SET price_ht = 2000 WHERE ref = $1`, p.Ref); err != nil {
		t.Fatal(err)
	}
	cart, err := m.GetCart(ctx, user)
	if err != nil || cart.TotalTTC != 2400 {
		t.Errorf("le panier doit suivre le prix actuel : %+v, err %v", cart, err)
	}
}
