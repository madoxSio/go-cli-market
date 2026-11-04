package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"time"
)

const (
	codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	codeLength   = 8
	codeTTL      = 15 * time.Minute
	sessionTTL   = 24 * time.Hour
)

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(email) < 3 || len(email) > 254 || strings.Count(email, "@") != 1 {
		return "", ValidationError{Message: "email invalide"}
	}
	local, domain, _ := strings.Cut(email, "@")
	if local == "" || !strings.Contains(domain, ".") || strings.ContainsAny(email, " \t") {
		return "", ValidationError{Message: "email invalide"}
	}
	return email, nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return ValidationError{Message: "le mot de passe doit contenir au moins 8 caractères"}
	}
	if len(password) > 72 {
		return ValidationError{Message: "le mot de passe dépasse 72 caractères"}
	}
	return nil
}

func normalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func codesMatch(got, want string) bool {
	if want == "" || len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func newCode() (string, error) {
	buf := make([]byte, codeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, codeLength)
	for i, b := range buf {
		out[i] = codeAlphabet[int(b)%len(codeAlphabet)]
	}
	return string(out), nil
}

func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
