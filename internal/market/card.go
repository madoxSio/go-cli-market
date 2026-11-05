package market

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Card regroupe ce que le client saisit pour payer. Elle n'est jamais enregistrée :
// seuls les 4 derniers chiffres du numéro sont conservés avec la commande.
type Card struct {
	Number string `json:"number"` // 13 à 19 chiffres, espaces et tirets acceptés
	Expiry string `json:"expiry"` // « MM/AA » ou « MM/AAAA »
	CVC    string `json:"cvc"`    // 3 chiffres
}

// ValidateCard vérifie la carte (algorithme de Luhn, date d'expiration, CVC)
// et renvoie les 4 derniers chiffres du numéro. now est l'heure actuelle, passée
// en paramètre pour que les tests puissent choisir la date.
func ValidateCard(c Card, now time.Time) (string, error) {
	number := strings.NewReplacer(" ", "", "-", "").Replace(c.Number)
	if len(number) < 13 || len(number) > 19 || !onlyDigits(number) || !luhnValid(number) {
		return "", fmt.Errorf("numéro de carte invalide : %w", ErrInvalidInput)
	}

	month, year, err := parseExpiry(c.Expiry)
	if err != nil {
		return "", err
	}
	// Une carte « 04/27 » reste valable jusqu'au dernier jour d'avril 2027 inclus.
	endOfValidity := time.Date(year, time.Month(month)+1, 1, 0, 0, 0, 0, now.Location())
	if !now.Before(endOfValidity) {
		return "", fmt.Errorf("carte expirée : %w", ErrInvalidInput)
	}

	if len(c.CVC) != 3 || !onlyDigits(c.CVC) {
		return "", fmt.Errorf("CVC invalide, 3 chiffres attendus : %w", ErrInvalidInput)
	}

	return number[len(number)-4:], nil
}

// luhnValid applique l'algorithme de Luhn : en partant de la droite, on double un chiffre
// sur deux (en retirant 9 si le résultat dépasse 9) ; la somme doit être un multiple de 10.
func luhnValid(digits string) bool {
	sum := 0
	double := false
	for i := len(digits) - 1; i >= 0; i-- {
		d := int(digits[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}

// parseExpiry lit « MM/AA » ou « MM/AAAA » et renvoie le mois (1 à 12) et l'année sur 4 chiffres.
func parseExpiry(s string) (month, year int, err error) {
	invalid := fmt.Errorf("date d'expiration invalide, format MM/AA attendu : %w", ErrInvalidInput)

	mm, yy, ok := strings.Cut(strings.TrimSpace(s), "/")
	if !ok || len(mm) != 2 || (len(yy) != 2 && len(yy) != 4) || !onlyDigits(mm) || !onlyDigits(yy) {
		return 0, 0, invalid
	}
	month, _ = strconv.Atoi(mm)
	year, _ = strconv.Atoi(yy)
	if month < 1 || month > 12 {
		return 0, 0, invalid
	}
	if len(yy) == 2 {
		year += 2000
	}
	return month, year, nil
}

// onlyDigits indique si s n'est fait que de chiffres de 0 à 9 (et n'est pas vide).
func onlyDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
