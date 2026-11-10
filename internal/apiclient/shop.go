package apiclient

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/madoxSio/go-cli-market/internal/api"
	"github.com/madoxSio/go-cli-market/internal/market"
)

// SearchProducts interroge GET /products. Un filtre vide renvoie tout le catalogue.
func (c *Client) SearchProducts(ctx context.Context, filter market.ProductFilter) ([]market.Product, error) {
	query := url.Values{}
	if text := strings.TrimSpace(filter.Query); text != "" {
		query.Set("q", text)
	}
	if category := strings.TrimSpace(filter.Category); category != "" {
		query.Set("category", category)
	}
	setCents(query, "min_price", filter.MinHT)
	setCents(query, "max_price", filter.MaxHT)
	setCents(query, "min_ttc", filter.MinTTC)
	setCents(query, "max_ttc", filter.MaxTTC)

	path := "/products"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var products []market.Product
	err := c.do(ctx, http.MethodGet, path, nil, &products, false)
	if products == nil {
		products = []market.Product{}
	}
	return products, err
}

// CreateProduct ajoute un produit. Réservé à un administrateur. La réponse contient la ref PDT-.
func (c *Client) CreateProduct(ctx context.Context, in market.ProductInput) (market.Product, error) {
	var product market.Product
	err := c.do(ctx, http.MethodPost, "/admin/products", in, &product, true)
	return product, err
}

// Cart lit le panier en cours du client connecté.
func (c *Client) Cart(ctx context.Context) (market.Cart, error) {
	var cart market.Cart
	err := c.do(ctx, http.MethodGet, "/cart", nil, &cart, true)
	return cart, err
}

// AddCartItems ajoute les quantités indiquées aux produits du panier.
func (c *Client) AddCartItems(ctx context.Context, items []api.ItemRequest) (market.Cart, error) {
	var cart market.Cart
	err := c.do(ctx, http.MethodPost, "/cart/items", api.CartItemsRequest{Items: items}, &cart, true)
	return cart, err
}

// SetCartItems remplace les quantités. Une quantité 0 retire le produit.
func (c *Client) SetCartItems(ctx context.Context, items []api.ItemRequest) (market.Cart, error) {
	var cart market.Cart
	err := c.do(ctx, http.MethodPatch, "/cart/items", api.CartItemsRequest{Items: items}, &cart, true)
	return cart, err
}

// RemoveCartItem retire un produit du panier.
func (c *Client) RemoveCartItem(ctx context.Context, productRef string) (market.Cart, error) {
	var cart market.Cart
	err := c.do(ctx, http.MethodDelete, "/cart/items/"+url.PathEscape(productRef), nil, &cart, true)
	return cart, err
}

// Checkout paie le panier en cours et renvoie la commande créée.
func (c *Client) Checkout(ctx context.Context, card market.Card) (market.Order, error) {
	var order market.Order
	err := c.do(ctx, http.MethodPost, "/cart/checkout", api.CheckoutRequest{Card: card}, &order, true)
	return order, err
}

// Orders liste les commandes du client connecté, la plus récente d'abord.
func (c *Client) Orders(ctx context.Context) ([]market.Order, error) {
	var orders []market.Order
	err := c.do(ctx, http.MethodGet, "/orders", nil, &orders, true)
	if orders == nil {
		orders = []market.Order{}
	}
	return orders, err
}

// AllOrders liste les commandes de tous les clients. Réservé à un administrateur.
func (c *Client) AllOrders(ctx context.Context) ([]market.Order, error) {
	var orders []market.Order
	err := c.do(ctx, http.MethodGet, "/admin/orders", nil, &orders, true)
	if orders == nil {
		orders = []market.Order{}
	}
	return orders, err
}

// CreateOrder crée une commande « en attente » pour un client. Réservé à un administrateur.
func (c *Client) CreateOrder(ctx context.Context, userID int64, items []api.ItemRequest) (market.Order, error) {
	var order market.Order
	req := api.CreateOrderRequest{UserID: userID, Items: items}
	err := c.do(ctx, http.MethodPost, "/admin/orders", req, &order, true)
	return order, err
}

// UpdateOrderStatus change le statut d'une commande. La raison est obligatoire pour une annulation.
func (c *Client) UpdateOrderStatus(ctx context.Context, ref string, status market.OrderStatus, reason string) (market.Order, error) {
	var order market.Order
	req := api.UpdateStatusRequest{Status: status, Reason: reason}
	err := c.do(ctx, http.MethodPatch, "/admin/orders/"+url.PathEscape(ref)+"/status", req, &order, true)
	return order, err
}

func setCents(query url.Values, key string, value *market.Cents) {
	if value == nil {
		return
	}
	query.Set(key, strconv.FormatInt(int64(*value), 10))
}
