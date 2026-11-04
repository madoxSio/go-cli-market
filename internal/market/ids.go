package market

import (
	"crypto/rand"
	"strings"
)

// Préfixes des identifiants métier.
const (
	PrefixProduct = "PDT"
	PrefixCart    = "BSK"
	PrefixOrder   = "CMD"
)

const (
	idAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	idLength   = 6
	// maxUnbiased est le plus grand multiple de 36 qui tient dans un octet (36 × 7 = 252).
	// Les octets de 252 à 255 sont rejetés, sinon les 4 premières lettres sortiraient plus souvent.
	maxUnbiased = 252
)

// NewBusinessID tire un identifiant métier aléatoire, par exemple « PDT-7D2K8N ».
// Il n'est pas garanti unique : la base le vérifie (colonne UNIQUE) et l'appelant retente en cas de collision.
func NewBusinessID(prefix string) string {
	var b strings.Builder
	b.Grow(len(prefix) + 1 + idLength)
	b.WriteString(prefix)
	b.WriteByte('-')

	buf := make([]byte, 1)
	for n := 0; n < idLength; {
		rand.Read(buf) // crypto/rand ne renvoie jamais d'erreur depuis Go 1.24
		if buf[0] >= maxUnbiased {
			continue
		}
		b.WriteByte(idAlphabet[buf[0]%byte(len(idAlphabet))])
		n++
	}
	return b.String()
}

// IsBusinessID indique si id a la forme « prefix-XXXXXX » avec des caractères de l'alphabet.
func IsBusinessID(id, prefix string) bool {
	code, ok := strings.CutPrefix(id, prefix+"-")
	if !ok || len(code) != idLength {
		return false
	}
	for _, r := range code {
		if !strings.ContainsRune(idAlphabet, r) {
			return false
		}
	}
	return true
}
