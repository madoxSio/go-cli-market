package market

import (
	"errors"
	"testing"
	"time"
)

func TestValidateCard(t *testing.T) {
	// Date fixe : le test donne le même résultat quel que soit le jour où on le lance.
	now := time.Date(2026, time.November, 5, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		card      Card
		wantLast4 string
		wantErr   bool
	}{
		{"visa de test", Card{"4242424242424242", "12/28", "123"}, "4242", false},
		{"mastercard de test", Card{"5555555555554444", "01/30", "000"}, "4444", false},
		{"amex 15 chiffres", Card{"378282246310005", "06/27", "999"}, "0005", false},
		{"avec espaces", Card{"4242 4242 4242 4242", "12/28", "123"}, "4242", false},
		{"avec tirets", Card{"4242-4242-4242-4242", "12/28", "123"}, "4242", false},
		{"année sur 4 chiffres", Card{"4242424242424242", "12/2028", "123"}, "4242", false},
		{"expire ce mois-ci", Card{"4242424242424242", "11/26", "123"}, "4242", false},

		{"luhn faux", Card{"4242424242424241", "12/28", "123"}, "", true},
		{"lettres", Card{"4242abcd42424242", "12/28", "123"}, "", true},
		{"trop court", Card{"424242424242", "12/28", "123"}, "", true},
		{"trop long", Card{"42424242424242424242", "12/28", "123"}, "", true},
		{"vide", Card{"", "12/28", "123"}, "", true},
		{"expirée le mois dernier", Card{"4242424242424242", "10/26", "123"}, "", true},
		{"expirée depuis longtemps", Card{"4242424242424242", "01/20", "123"}, "", true},
		{"mois 13", Card{"4242424242424242", "13/28", "123"}, "", true},
		{"mois 00", Card{"4242424242424242", "00/28", "123"}, "", true},
		{"date sans barre", Card{"4242424242424242", "1228", "123"}, "", true},
		{"date sur 1 chiffre", Card{"4242424242424242", "1/28", "123"}, "", true},
		{"cvc 2 chiffres", Card{"4242424242424242", "12/28", "12"}, "", true},
		{"cvc 4 chiffres", Card{"4242424242424242", "12/28", "1234"}, "", true},
		{"cvc lettres", Card{"4242424242424242", "12/28", "12a"}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			last4, err := ValidateCard(tt.card, now)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf("erreur = %v, attendu ErrInvalidInput", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("erreur inattendue : %v", err)
			}
			if last4 != tt.wantLast4 {
				t.Errorf("4 derniers chiffres = %q, attendu %q", last4, tt.wantLast4)
			}
		})
	}
}

func TestLuhnValid(t *testing.T) {
	tests := []struct {
		digits string
		want   bool
	}{
		{"79927398713", true}, // exemple classique de l'algorithme
		{"79927398710", false},
		{"0", true},
		{"18", true}, // 1×2 + 8 = 10
		{"19", false},
	}
	for _, tt := range tests {
		if got := luhnValid(tt.digits); got != tt.want {
			t.Errorf("luhnValid(%q) = %v, attendu %v", tt.digits, got, tt.want)
		}
	}
}
