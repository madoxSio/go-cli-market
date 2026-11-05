package market

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// OrderStatus est l'état d'une commande. Sa valeur est celle stockée en base et échangée en JSON.
type OrderStatus string

// Statuts possibles d'une commande.
const (
	StatusPending   OrderStatus = "pending"
	StatusPaid      OrderStatus = "paid"
	StatusShipping  OrderStatus = "shipping"
	StatusDelivered OrderStatus = "delivered"
	StatusCancelled OrderStatus = "cancelled"
)

// Order est une commande : les lignes du panier au moment du paiement (prix figés),
// le statut et, si elle a été payée par carte, les 4 derniers chiffres de la carte.
type Order struct {
	Ref          string      `json:"ref"`
	UserID       int64       `json:"user_id"`
	UserEmail    string      `json:"user_email,omitempty"`
	Status       OrderStatus `json:"status"`
	CancelReason string      `json:"cancel_reason,omitempty"`
	Items        []Item      `json:"items"`
	TotalTTC     Cents       `json:"total_ttc"`
	CardLast4    string      `json:"card_last4,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// transitions liste, pour chaque statut, ceux vers lesquels une commande peut passer.
// Un statut absent de la map (livrée, annulée) est final : plus aucun changement possible.
var transitions = map[OrderStatus][]OrderStatus{
	StatusPending:  {StatusPaid, StatusCancelled},
	StatusPaid:     {StatusShipping, StatusCancelled},
	StatusShipping: {StatusDelivered},
}

// Valid indique si s est l'un des cinq statuts connus.
func (s OrderStatus) Valid() bool {
	switch s {
	case StatusPending, StatusPaid, StatusShipping, StatusDelivered, StatusCancelled:
		return true
	default:
		return false
	}
}

// Label renvoie le libellé français du statut, pour l'affichage.
func (s OrderStatus) Label() string {
	switch s {
	case StatusPending:
		return "en attente"
	case StatusPaid:
		return "payée"
	case StatusShipping:
		return "en cours de livraison"
	case StatusDelivered:
		return "livrée"
	case StatusCancelled:
		return "annulée"
	default:
		return string(s)
	}
}

// CanTransition indique si une commande peut passer du statut from au statut to.
func CanTransition(from, to OrderStatus) bool {
	return slices.Contains(transitions[from], to)
}

// ValidateTransition vérifie un changement de statut demandé par un administrateur.
// Une annulation exige une raison.
func ValidateTransition(from, to OrderStatus, reason string) error {
	if !to.Valid() {
		return fmt.Errorf("statut %q inconnu : %w", to, ErrInvalidInput)
	}
	if !CanTransition(from, to) {
		return fmt.Errorf("une commande %s ne peut pas devenir %s : %w", from.Label(), to.Label(), ErrConflict)
	}
	if to == StatusCancelled && strings.TrimSpace(reason) == "" {
		return fmt.Errorf("la raison de l'annulation est obligatoire : %w", ErrInvalidInput)
	}
	return nil
}
