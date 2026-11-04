package auth

import (
	"context"
	"errors"
	"testing"
)

func TestAdminManagesUsers(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	if err := svc.EnsureAdmin(ctx, "Admin@Market.local", "adminadmin"); err != nil {
		t.Fatal(err)
	}
	if err := svc.EnsureAdmin(ctx, "autre@market.local", "autre-mot-de-passe"); err != nil {
		t.Fatal(err)
	}
	users, err := svc.ListUsers(ctx)
	if err != nil || len(users) != 1 || users[0].Email != "admin@market.local" || !users[0].Confirmed {
		t.Fatalf("admins = %+v err %v", users, err)
	}
	admin := users[0]

	created, err := svc.CreateUser(ctx, "client@example.com", "motdepasse", "client", false)
	if err != nil || created.Confirmed || created.Role != "client" {
		t.Fatalf("création = %+v err %v", created, err)
	}
	confirmed, err := svc.ConfirmUser(ctx, created.ID)
	if err != nil || !confirmed.Confirmed {
		t.Fatalf("confirmation = %+v err %v", confirmed, err)
	}
	renamed, err := svc.UpdateUser(ctx, created.ID, "client2@example.com", "", "client")
	if err != nil || renamed.Email != "client2@example.com" {
		t.Fatalf("email = %+v err %v", renamed, err)
	}
	if _, _, err := svc.Login(ctx, "client2@example.com", "motdepasse"); err != nil {
		t.Fatalf("login après modification : %v", err)
	}
	if err := svc.DeleteUser(ctx, admin.ID, admin.ID); !errors.Is(err, ErrSelfDelete) {
		t.Fatalf("auto-suppression = %v", err)
	}
	if err := svc.DeleteUser(ctx, admin.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteUser(ctx, 0, admin.ID); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("dernier admin = %v", err)
	}
}
