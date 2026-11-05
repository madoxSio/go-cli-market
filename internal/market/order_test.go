package market

import (
	"errors"
	"testing"
)

var allStatuses = []OrderStatus{StatusPending, StatusPaid, StatusShipping, StatusDelivered, StatusCancelled}

func TestCanTransition(t *testing.T) {
	// Les seules transitions autorisées. Toutes les autres paires doivent être refusées.
	allowed := map[[2]OrderStatus]bool{
		{StatusPending, StatusPaid}:       true,
		{StatusPending, StatusCancelled}:  true,
		{StatusPaid, StatusShipping}:      true,
		{StatusPaid, StatusCancelled}:     true,
		{StatusShipping, StatusDelivered}: true,
	}

	// 5 × 5 = 25 paires, y compris « rester dans le même statut ».
	for _, from := range allStatuses {
		for _, to := range allStatuses {
			want := allowed[[2]OrderStatus{from, to}]
			if got := CanTransition(from, to); got != want {
				t.Errorf("CanTransition(%s, %s) = %v, attendu %v", from, to, got, want)
			}
		}
	}
}

func TestCanTransitionUnknownStatus(t *testing.T) {
	if CanTransition("inconnu", StatusPaid) {
		t.Error("une transition depuis un statut inconnu ne doit pas être autorisée")
	}
	if CanTransition(StatusPending, "inconnu") {
		t.Error("une transition vers un statut inconnu ne doit pas être autorisée")
	}
}

func TestValidateTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    OrderStatus
		to      OrderStatus
		reason  string
		wantErr error
	}{
		{"paiement", StatusPending, StatusPaid, "", nil},
		{"expédition", StatusPaid, StatusShipping, "", nil},
		{"livraison", StatusShipping, StatusDelivered, "", nil},
		{"annulation avec raison", StatusPaid, StatusCancelled, "rupture de stock", nil},
		{"annulation sans raison", StatusPending, StatusCancelled, "", ErrInvalidInput},
		{"annulation raison vide", StatusPaid, StatusCancelled, "   ", ErrInvalidInput},
		{"statut inconnu", StatusPending, "perdue", "", ErrInvalidInput},
		{"retour en arrière", StatusShipping, StatusPaid, "", ErrConflict},
		{"annuler une livraison", StatusDelivered, StatusCancelled, "trop tard", ErrConflict},
		{"sauter une étape", StatusPending, StatusDelivered, "", ErrConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransition(tt.from, tt.to, tt.reason)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("erreur inattendue : %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("erreur = %v, attendu %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderStatusValidAndLabel(t *testing.T) {
	for _, s := range allStatuses {
		if !s.Valid() {
			t.Errorf("%s devrait être valide", s)
		}
		if s.Label() == string(s) {
			t.Errorf("%s n'a pas de libellé français", s)
		}
	}
	if OrderStatus("perdue").Valid() {
		t.Error("« perdue » ne devrait pas être un statut valide")
	}
}
