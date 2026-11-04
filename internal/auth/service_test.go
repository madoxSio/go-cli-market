package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	svc, err := NewService(NewMemoryRepository())
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestRegisterConfirmLoginAndReset(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	const email = "Ada@Example.com"
	const password = "motdepasse"

	code, err := svc.Register(ctx, email, password)
	if err != nil {
		t.Fatalf("register : %v", err)
	}
	if len(code) != codeLength {
		t.Fatalf("code = %q, longueur %d attendue", code, codeLength)
	}

	if _, _, err := svc.Login(ctx, email, password); !errors.Is(err, ErrNotConfirmed) {
		t.Fatalf("login avant confirmation = %v, attendu %v", err, ErrNotConfirmed)
	}
	if err := svc.Confirm(ctx, email, "mauvais"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("mauvais code = %v", err)
	}
	if err := svc.Confirm(ctx, "ada@example.com", code); err != nil {
		t.Fatalf("confirm : %v", err)
	}
	if err := svc.Confirm(ctx, email, code); !errors.Is(err, ErrAlreadyConfirmed) {
		t.Fatalf("deuxième confirmation = %v", err)
	}

	token, user, err := svc.Login(ctx, email, password)
	if err != nil {
		t.Fatalf("login : %v", err)
	}
	if user.Email != "ada@example.com" || user.Role != "client" || token == "" {
		t.Fatalf("session inattendue : %+v token %q", user, token)
	}
	got, err := svc.CurrentUser(ctx, token)
	if err != nil || got.ID != user.ID {
		t.Fatalf("session = %+v err %v", got, err)
	}

	reset, err := svc.RequestPasswordReset(ctx, email)
	if err != nil || reset == "" {
		t.Fatalf("reset code = %q err %v", reset, err)
	}
	if err := svc.ResetPassword(ctx, email, reset, "nouveau-mot"); err != nil {
		t.Fatalf("reset : %v", err)
	}
	if _, err := svc.CurrentUser(ctx, token); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("ancienne session = %v, attendu non authentifié", err)
	}
	if _, _, err := svc.Login(ctx, email, password); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("ancien mot de passe = %v", err)
	}
	if _, _, err := svc.Login(ctx, email, "nouveau-mot"); err != nil {
		t.Fatalf("nouveau mot de passe : %v", err)
	}
}

func TestConfirmationExpires(t *testing.T) {
	svc := newTestService(t)
	start := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return start }

	code, err := svc.Register(context.Background(), "a@b.co", "motdepasse")
	if err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return start.Add(16 * time.Minute) }
	if err := svc.Confirm(context.Background(), "a@b.co", code); !errors.Is(err, ErrExpiredCode) {
		t.Fatalf("code expiré = %v", err)
	}
}

func TestUnknownEmailDoesNotRevealAccount(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	if _, _, err := svc.Login(ctx, "absent@example.com", "motdepasse"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("login inconnu = %v", err)
	}
	code, err := svc.RequestPasswordReset(ctx, "absent@example.com")
	if err != nil || code != "" {
		t.Fatalf("reset inconnu = %q err %v", code, err)
	}
	if _, err := svc.Register(ctx, "pas-un-email", "motdepasse"); err == nil {
		t.Fatal("email invalide accepté")
	}
	if _, err := svc.Register(ctx, "a@b.co", "court"); err == nil {
		t.Fatal("mot de passe court accepté")
	}
}

func TestRenewConfirmation(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	first, err := svc.Register(ctx, "a@b.co", "motdepasse")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.RenewConfirmation(ctx, "a@b.co")
	if err != nil || second == "" || second == first {
		t.Fatalf("nouveau code = %q err %v (ancien %q)", second, err, first)
	}
	if err := svc.Confirm(ctx, "a@b.co", first); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("ancien code = %v", err)
	}
	if err := svc.Confirm(ctx, "a@b.co", strings.ToLower(second)); err != nil {
		t.Fatalf("nouveau code en minuscules : %v", err)
	}
}
