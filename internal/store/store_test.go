package store

import (
	"database/sql"
	"os"
	"testing"
)

// openTestDB ouvre la base de test et applique le schéma. Sans TEST_DATABASE_URL, le test est ignoré.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL non défini")
	}
	db, err := Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMigrateTwice(t *testing.T) {
	db := openTestDB(t)

	// Une deuxième migration ne doit rien casser : le script est idempotent.
	if err := Migrate(db); err != nil {
		t.Fatalf("deuxième migration : %v", err)
	}

	for _, table := range []string{"users", "sessions", "products", "carts", "cart_items", "orders", "order_items"} {
		var exists bool
		err := db.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, "public."+table).Scan(&exists)
		if err != nil || !exists {
			t.Errorf("table %s absente (err %v)", table, err)
		}
	}
}
