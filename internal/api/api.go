// Package api décrit les corps JSON des requêtes de l'API qui ne sont pas déjà des types
// de internal/market. Il est partagé par le serveur (internal/httpapi) et les clients
// (internal/apiclient) : un seul endroit à modifier si le format change.
// La liste des routes est dans README.md, à côté de ce fichier.
package api

import "github.com/madoxSio/go-cli-market/internal/market"

// ItemRequest désigne un produit et une quantité.
type ItemRequest struct {
	ProductRef string `json:"product_ref"`
	Quantity   int    `json:"quantity"`
}

// CartItemsRequest est le corps de POST /cart/items (ajouter) et PATCH /cart/items (changer les quantités).
// Il contient un ou plusieurs produits.
type CartItemsRequest struct {
	Items []ItemRequest `json:"items"`
}

// CheckoutRequest est le corps de POST /cart/checkout.
type CheckoutRequest struct {
	Card market.Card `json:"card"`
}

// CreateOrderRequest est le corps de POST /admin/orders : une commande créée pour un client.
type CreateOrderRequest struct {
	UserID int64         `json:"user_id"`
	Items  []ItemRequest `json:"items"`
}

// UpdateStatusRequest est le corps de PATCH /admin/orders/{ref}/status.
// Reason est obligatoire quand Status vaut "cancelled".
type UpdateStatusRequest struct {
	Status market.OrderStatus `json:"status"`
	Reason string             `json:"reason,omitempty"`
}

// ErrorResponse est le corps de toutes les réponses d'erreur (codes 4xx et 5xx).
type ErrorResponse struct {
	Error string `json:"error"`
}
