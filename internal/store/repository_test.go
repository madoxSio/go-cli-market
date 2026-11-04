package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/madoxSio/go-cli-market/internal/auth"
)

func TestPostgresRepository(t *testing.T) {
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

	repo := NewRepository(db)
	ctx := context.Background()
	email := "sql-" + time.Now().Format("150405.000") + "@example.com"
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM users WHERE email = $1`, email)
	})

	created, err := repo.CreateAccount(ctx, auth.Account{
		Email:               email,
		PasswordHash:        "hash",
		Role:                "client",
		ConfirmationCode:    "ABCD2345",
		ConfirmationExpires: time.Now().Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateAccount(ctx, created); err != auth.ErrEmailTaken {
		t.Fatalf("doublon = %v", err)
	}

	created.Confirmed = true
	created.ConfirmationCode = ""
	created.ConfirmationExpires = time.Time{}
	if err := repo.UpdateAccount(ctx, created); err != nil {
		t.Fatal(err)
	}
	found, err := repo.FindByEmail(ctx, email)
	if err != nil || !found.Confirmed || found.ConfirmationCode != "" {
		t.Fatalf("compte relu = %+v err %v", found, err)
	}

	expires := time.Now().Add(time.Hour)
	if err := repo.CreateSession(ctx, found.ID, "jeton-test", expires); err != nil {
		t.Fatal(err)
	}
	byToken, err := repo.FindBySession(ctx, "jeton-test", time.Now())
	if err != nil || byToken.Email != email {
		t.Fatalf("session = %+v err %v", byToken, err)
	}
	if err := repo.DeleteUserSessions(ctx, found.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.FindBySession(ctx, "jeton-test", time.Now()); err != auth.ErrNotFound {
		t.Fatalf("session supprimée = %v", err)
	}
}
