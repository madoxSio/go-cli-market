package auth

import "time"

// User est la vue publique d'un compte, sans empreinte de mot de passe.
type User struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// Account est le compte complet manipulé par le service et le dépôt.
type Account struct {
	ID                  int64
	Email               string
	PasswordHash        string
	Role                string
	Confirmed           bool
	ConfirmationCode    string
	ConfirmationExpires time.Time
	ResetCode           string
	ResetExpires        time.Time
}

// Public retire les secrets avant de répondre au client.
func (a Account) Public() User {
	return User{ID: a.ID, Email: a.Email, Role: a.Role}
}
