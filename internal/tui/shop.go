package tui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/madoxSio/go-cli-market/internal/api"
	"github.com/madoxSio/go-cli-market/internal/market"
)

type section int

const (
	sectionHome section = iota
	sectionProducts
	sectionProduct
	sectionCart
	sectionOrders
)

const (
	actionSearch action = iota + 20
	actionAdd
	actionSetQty
	actionRemove
	actionCheckout
	actionLoadCart
	actionLoadOrders
)

type shopFields struct {
	query, category     string
	minHT, maxHT        string
	minTTC, maxTTC      string
	quantity            string
	number, expiry, cvc string
}

func (m *model) activateShop(id string) (tea.Model, tea.Cmd) {
	switch id {
	case "search":
		m.status = ""
		return m.openShopForm(actionSearch)
	case "cart":
		return m.loadCart()
	case "orders":
		return m.loadOrders()
	case "home":
		m.section = sectionHome
		m.screen = screenMenu
		m.useSessionMenu()
		return *m, nil
	case "pay":
		m.shop.number = ""
		m.shop.expiry = ""
		m.shop.cvc = ""
		return m.openShopForm(actionCheckout)
	case "add":
		m.shop.quantity = "1"
		return m.openShopForm(actionAdd)
	case "qty":
		m.shop.quantity = strconv.Itoa(m.lineQuantity())
		return m.openShopForm(actionSetQty)
	case "remove":
		m.pending = actionRemove
		m.screen = screenLoading
		m.loadingText = "Retrait du produit..."
		return *m, tea.Batch(m.spinner.Tick, m.submit())
	case "back-products":
		m.useProductList()
		return *m, nil
	case "back-cart":
		m.useCartMenu()
		return *m, nil
	default:
		if ref, ok := strings.CutPrefix(id, "product:"); ok {
			for _, product := range m.products {
				if product.Ref == ref {
					m.product = product
					m.useProductMenu()
					return *m, nil
				}
			}
		}
		if ref, ok := strings.CutPrefix(id, "item:"); ok {
			m.lineRef = ref
			m.useLineMenu()
			return *m, nil
		}
		if ref, ok := strings.CutPrefix(id, "order:"); ok {
			for _, order := range m.orders {
				if order.Ref == ref {
					m.showOrder(order)
					return *m, nil
				}
			}
		}
		return *m, nil
	}
}

func (m *model) openShopForm(kind action) (tea.Model, tea.Cmd) {
	m.pending = kind
	m.formTitle = shopTitle(kind)
	m.form = m.buildShopForm(kind)
	m.screen = screenForm
	return *m, m.form.Init()
}

func (m *model) loadCart() (tea.Model, tea.Cmd) {
	m.pending = actionLoadCart
	m.screen = screenLoading
	m.loadingText = "Chargement du panier..."
	return *m, tea.Batch(m.spinner.Tick, m.submit())
}

func (m *model) loadOrders() (tea.Model, tea.Cmd) {
	m.pending = actionLoadOrders
	m.screen = screenLoading
	m.loadingText = "Chargement des commandes..."
	return *m, tea.Batch(m.spinner.Tick, m.submit())
}

func (m *model) leaveForm() (tea.Model, tea.Cmd) {
	switch m.pending {
	case actionAdd:
		m.useProductMenu()
	case actionSetQty, actionCheckout:
		m.useCartMenu()
	case actionSearch:
		switch m.section {
		case sectionProducts:
			m.useProductList()
		case sectionCart:
			m.useCartMenu()
		default:
			m.backHome()
		}
	default:
		m.backHome()
	}
	return *m, nil
}

func (m *model) backHome() {
	m.section = sectionHome
	m.screen = screenMenu
	if m.authed {
		m.useSessionMenu()
		return
	}
	m.useGuestMenu()
}

func (m *model) goBack() (tea.Model, tea.Cmd) {
	switch m.section {
	case sectionProduct:
		m.useProductList()
	case sectionOrders:
		if m.screen == screenDetail {
			m.useOrderList()
			return *m, nil
		}
		m.backHome()
	default:
		m.backHome()
	}
	return *m, nil
}

func (m *model) finishShop(msg doneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status = msg.err.Error()
		m.statusOK = false
		switch msg.kind {
		case actionSearch, actionAdd, actionSetQty, actionCheckout:
			return m.openShopForm(msg.kind)
		case actionRemove:
			m.useCartMenu()
		default:
			m.backHome()
		}
		return *m, nil
	}

	switch msg.kind {
	case actionSearch:
		m.products = msg.products
		m.statusOK = true
		if len(msg.products) == 0 {
			m.status = "Aucun produit ne correspond."
		} else if len(msg.products) == 1 {
			m.status = "1 produit"
		} else {
			m.status = fmt.Sprintf("%d produits", len(msg.products))
		}
		m.useProductList()
	case actionAdd:
		m.cart = msg.cart
		m.status = "Ajouté au panier"
		m.statusOK = true
		m.useCartMenu()
	case actionSetQty:
		m.cart = msg.cart
		m.status = "Quantité mise à jour"
		m.statusOK = true
		m.useCartMenu()
	case actionRemove, actionLoadCart:
		m.cart = msg.cart
		if msg.kind == actionRemove {
			m.status = "Produit retiré"
			m.statusOK = true
		}
		m.useCartMenu()
	case actionCheckout:
		m.order = msg.order
		m.orders = append([]market.Order{msg.order}, m.orders...)
		m.cart = market.Cart{}
		m.status = "Paiement accepté"
		m.statusOK = true
		m.showOrder(msg.order)
	case actionLoadOrders:
		m.orders = msg.orders
		m.useOrderList()
	default:
		m.backHome()
	}
	return *m, nil
}

func (m *model) submitShop() tea.Cmd {
	kind := m.pending
	fields := *m.shop
	productRef := m.product.Ref
	lineRef := m.lineRef
	client := m.api
	return func() tea.Msg {
		ctx := context.Background()
		switch kind {
		case actionSearch:
			filter, err := searchFilter(fields)
			if err != nil {
				return doneMsg{kind: kind, err: err}
			}
			products, err := client.SearchProducts(ctx, filter)
			return doneMsg{kind: kind, products: products, err: err}
		case actionAdd:
			qty, err := parseQuantity(fields.quantity)
			if err != nil {
				return doneMsg{kind: kind, err: err}
			}
			cart, err := client.AddCartItems(ctx, []api.ItemRequest{{ProductRef: productRef, Quantity: qty}})
			return doneMsg{kind: kind, cart: cart, err: err}
		case actionSetQty:
			qty, err := parseQuantity(fields.quantity)
			if err != nil {
				return doneMsg{kind: kind, err: err}
			}
			cart, err := client.SetCartItems(ctx, []api.ItemRequest{{ProductRef: lineRef, Quantity: qty}})
			return doneMsg{kind: kind, cart: cart, err: err}
		case actionRemove:
			cart, err := client.RemoveCartItem(ctx, lineRef)
			return doneMsg{kind: kind, cart: cart, err: err}
		case actionCheckout:
			order, err := client.Checkout(ctx, market.Card{Number: fields.number, Expiry: fields.expiry, CVC: fields.cvc})
			return doneMsg{kind: kind, order: order, err: err}
		case actionLoadCart:
			cart, err := client.Cart(ctx)
			return doneMsg{kind: kind, cart: cart, err: err}
		case actionLoadOrders:
			orders, err := client.Orders(ctx)
			return doneMsg{kind: kind, orders: orders, err: err}
		default:
			return doneMsg{kind: kind, err: errors.New("action inconnue")}
		}
	}
}

func (m *model) buildShopForm(kind action) *huh.Form {
	switch kind {
	case actionSearch:
		return m.finishForm(huh.NewGroup(
			huh.NewInput().Title("Nom ou description").Placeholder("clavier").Value(&m.shop.query),
			huh.NewInput().Title("Catégorie").Placeholder("informatique").Value(&m.shop.category),
			huh.NewInput().Title("Prix HT minimum").Description("En euros, exemple 10 ou 10,50. Vide pour ignorer.").Value(&m.shop.minHT).Validate(optionalEuros),
			huh.NewInput().Title("Prix HT maximum").Description("En euros. Vide pour ignorer.").Value(&m.shop.maxHT).Validate(optionalEuros),
			huh.NewInput().Title("Prix TTC minimum").Description("En euros. Vide pour ignorer.").Value(&m.shop.minTTC).Validate(optionalEuros),
			huh.NewInput().Title("Prix TTC maximum").Description("En euros. Vide pour ignorer.").Value(&m.shop.maxTTC).Validate(optionalEuros),
		))
	case actionAdd, actionSetQty:
		return m.finishForm(huh.NewGroup(
			huh.NewInput().
				Title("Quantité").
				Description("De 1 à 99.").
				Value(&m.shop.quantity).
				Validate(func(value string) error {
					_, err := parseQuantity(value)
					return err
				}),
		))
	case actionCheckout:
		return m.finishForm(huh.NewGroup(
			huh.NewInput().
				Title("Numéro de carte").
				Placeholder("4242 4242 4242 4242").
				Value(&m.shop.number).
				Validate(validateCardNumber),
			huh.NewInput().
				Title("Expiration").
				Placeholder("MM/AA").
				CharLimit(7).
				Value(&m.shop.expiry).
				Validate(validateExpiry),
			huh.NewInput().
				Title("CVC").
				Placeholder("123").
				CharLimit(3).
				EchoMode(huh.EchoModePassword).
				Value(&m.shop.cvc).
				Validate(validateCVC),
		))
	default:
		return m.finishForm(huh.NewGroup(huh.NewInput().Title("Quantité").Value(&m.shop.quantity)))
	}
}

func (m *model) useProductList() {
	items := []menuItem{{id: "search", title: "Nouvelle recherche", desc: "Nom, catégorie ou prix"}}
	for _, product := range m.products {
		items = append(items, menuItem{
			id:    "product:" + product.Ref,
			title: product.Name,
			desc:  product.PriceTTC.String() + " TTC · " + product.Category,
		})
	}
	items = append(items, menuItem{id: "home", title: "Accueil", desc: "Retour au menu"})
	m.section = sectionProducts
	m.screen = screenMenu
	m.setMenu(items)
}

func (m *model) useProductMenu() {
	m.section = sectionProduct
	m.screen = screenMenu
	m.setMenu([]menuItem{
		{id: "add", title: "Ajouter au panier", desc: "Choisir la quantité"},
		{id: "back-products", title: "Retour", desc: "Liste des produits"},
	})
}

func (m *model) useCartMenu() {
	items := []menuItem{}
	if len(m.cart.Items) > 0 {
		items = append(items, menuItem{id: "pay", title: "Payer", desc: m.cart.TotalTTC.String() + " TTC · livraison offerte"})
	}
	for _, item := range m.cart.Items {
		items = append(items, menuItem{
			id:    "item:" + item.ProductRef,
			title: item.Name,
			desc:  fmt.Sprintf("%d × %s = %s", item.Quantity, item.UnitPriceTTC, item.LineTTC),
		})
	}
	if len(m.cart.Items) == 0 {
		items = append(items, menuItem{id: "search", title: "Chercher un produit", desc: "Le panier est vide"})
	}
	items = append(items, menuItem{id: "home", title: "Accueil", desc: "Retour au menu"})
	m.section = sectionCart
	m.screen = screenMenu
	m.setMenu(items)
}

func (m *model) useLineMenu() {
	m.section = sectionCart
	m.screen = screenMenu
	title := m.lineRef
	for _, item := range m.cart.Items {
		if item.ProductRef == m.lineRef {
			title = item.Name
		}
	}
	m.setMenu([]menuItem{
		{id: "qty", title: "Changer la quantité", desc: title},
		{id: "remove", title: "Retirer", desc: "Enlever ce produit du panier"},
		{id: "back-cart", title: "Retour", desc: "Voir le panier"},
	})
}

func (m *model) useOrderList() {
	items := make([]menuItem, 0, len(m.orders)+1)
	for _, order := range m.orders {
		items = append(items, menuItem{
			id:    "order:" + order.Ref,
			title: order.Ref,
			desc:  order.Status.Label() + " · " + order.TotalTTC.String() + " · " + order.CreatedAt.Local().Format("02/01/2006"),
		})
	}
	if len(items) == 0 {
		items = append(items, menuItem{id: "home", title: "Aucune commande", desc: "Retour au menu"})
	} else {
		items = append(items, menuItem{id: "home", title: "Accueil", desc: "Retour au menu"})
	}
	m.section = sectionOrders
	m.screen = screenMenu
	m.setMenu(items)
}

func (m *model) showOrder(order market.Order) {
	m.order = order
	m.section = sectionOrders
	m.screen = screenDetail
	m.detailTitle = order.Ref
	m.detail.Width = m.contentWidth()
	m.detail.Height = m.helpHeight()
	m.detail.SetContent(formatOrder(order))
	m.detail.GotoTop()
}

func (m model) shopHeading() (string, bool) {
	switch m.section {
	case sectionProducts:
		return "Catalogue", true
	case sectionProduct:
		if m.product.Name != "" {
			return m.product.Name, true
		}
		return "Produit", true
	case sectionCart:
		if m.cart.Ref != "" {
			return m.cart.Ref, true
		}
		return "Panier", true
	case sectionOrders:
		if m.screen == screenDetail && m.detailTitle != "" {
			return m.detailTitle, true
		}
		return "Commandes", true
	default:
		return "", false
	}
}

func (m model) shopTagline() (string, bool) {
	switch m.section {
	case sectionProducts:
		return "Recherche par nom, description, catégorie ou prix", true
	case sectionProduct:
		return m.product.Ref + " · " + m.product.PriceTTC.String() + " TTC", true
	case sectionCart:
		if len(m.cart.Items) == 0 {
			return "Panier vide · livraison offerte", true
		}
		return "Total " + m.cart.TotalTTC.String() + " TTC · livraison offerte", true
	case sectionOrders:
		if m.screen == screenDetail {
			return m.order.Status.Label(), true
		}
		return "La plus récente en premier", true
	default:
		return "", false
	}
}

func (m model) shopIntro() string {
	switch m.pending {
	case actionAdd:
		return m.product.Ref + " · " + m.product.PriceTTC.String() + " TTC\n" + m.product.Description
	case actionCheckout:
		if m.cart.Ref == "" {
			return "Livraison offerte."
		}
		return m.cart.Ref + " · " + m.cart.TotalTTC.String() + " TTC · livraison offerte"
	default:
		return ""
	}
}

func (m model) lineQuantity() int {
	for _, item := range m.cart.Items {
		if item.ProductRef == m.lineRef {
			return item.Quantity
		}
	}
	return 1
}

func formatOrder(order market.Order) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Statut     %s\n", order.Status.Label())
	if order.CancelReason != "" {
		fmt.Fprintf(&b, "Raison     %s\n", order.CancelReason)
	}
	fmt.Fprintf(&b, "Date       %s\n", order.CreatedAt.Local().Format("02/01/2006 15:04"))
	fmt.Fprintf(&b, "Total TTC  %s\n", order.TotalTTC)
	b.WriteString("Livraison  offerte\n")
	if order.CardLast4 != "" {
		fmt.Fprintf(&b, "Carte      •••• %s\n", order.CardLast4)
	}
	b.WriteString("\n")
	if len(order.Items) == 0 {
		b.WriteString("Aucun produit.\n")
		return b.String()
	}
	for _, item := range order.Items {
		fmt.Fprintf(&b, "%s\n  %s · %d × %s = %s\n", item.Name, item.ProductRef, item.Quantity, item.UnitPriceTTC, item.LineTTC)
	}
	return b.String()
}

func shopTitle(kind action) string {
	switch kind {
	case actionSearch:
		return "Rechercher un produit"
	case actionAdd:
		return "Ajouter au panier"
	case actionSetQty:
		return "Quantité"
	case actionCheckout:
		return "Paiement"
	default:
		return ""
	}
}

func isShop(kind action) bool {
	switch kind {
	case actionSearch, actionAdd, actionSetQty, actionRemove, actionCheckout, actionLoadCart, actionLoadOrders:
		return true
	default:
		return false
	}
}

func searchFilter(fields shopFields) (market.ProductFilter, error) {
	filter := market.ProductFilter{
		Query:    strings.TrimSpace(fields.query),
		Category: strings.TrimSpace(fields.category),
	}
	var err error
	if filter.MinHT, err = optionalCents(fields.minHT); err != nil {
		return filter, err
	}
	if filter.MaxHT, err = optionalCents(fields.maxHT); err != nil {
		return filter, err
	}
	if filter.MinTTC, err = optionalCents(fields.minTTC); err != nil {
		return filter, err
	}
	if filter.MaxTTC, err = optionalCents(fields.maxTTC); err != nil {
		return filter, err
	}
	return filter, nil
}

func optionalCents(raw string) (*market.Cents, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	cents, err := market.ParseEuros(raw)
	if err != nil {
		return nil, err
	}
	return &cents, nil
}

func optionalEuros(value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	_, err := market.ParseEuros(value)
	return err
}

func parseQuantity(raw string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 1 || n > market.MaxQuantity {
		return 0, fmt.Errorf("la quantité doit être entre 1 et %d", market.MaxQuantity)
	}
	return n, nil
}

func validateCardNumber(value string) error {
	_, err := market.ValidateCard(market.Card{Number: value, Expiry: "12/99", CVC: "000"}, time.Now())
	if err != nil {
		return errors.New("numéro de carte invalide")
	}
	return nil
}

func validateExpiry(value string) error {
	_, err := market.ValidateCard(market.Card{Number: "4242424242424242", Expiry: value, CVC: "000"}, time.Now())
	if err != nil {
		return errors.New("date invalide, format MM/AA")
	}
	return nil
}

func validateCVC(value string) error {
	if len(value) != 3 || strings.Trim(value, "0123456789") != "" {
		return errors.New("3 chiffres")
	}
	return nil
}
