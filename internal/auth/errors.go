package auth

import "errors"

// Erreurs métier renvoyées par le service. Les handlers HTTP les traduisent en codes de statut.
var (
	ErrEmailTaken         = errors.New("un compte existe déjà avec cet email")
	ErrInvalidCredentials = errors.New("email ou mot de passe incorrect")
	ErrNotConfirmed       = errors.New("le compte n'est pas confirmé")
	ErrInvalidCode        = errors.New("code invalide")
	ErrExpiredCode        = errors.New("code expiré")
	ErrAlreadyConfirmed   = errors.New("le compte est déjà confirmé")
	ErrNotFound           = errors.New("introuvable")
	ErrUnauthenticated    = errors.New("authentification requise")
)

// ValidationError signale une saisie refusée avant tout accès à la base.
type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}
