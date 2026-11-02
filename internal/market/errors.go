package market

import "errors"


var (
	ErrNotFound     = errors.New("introuvable")
	ErrInvalidInput = errors.New("données invalides")
	ErrConflict     = errors.New("conflit avec l'état actuel")
	ErrUnauthorized = errors.New("authentification requise")
	ErrForbidden    = errors.New("accès interdit")
)