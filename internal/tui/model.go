package tui

import (
	"context"
	"errors"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/madoxSio/go-cli-market/internal/apiclient"
	"github.com/madoxSio/go-cli-market/internal/frame"
	"github.com/madoxSio/go-cli-market/internal/market"
)

type screen int

const (
	screenMenu screen = iota
	screenForm
	screenNotice
	screenHelp
	screenLoading
	screenDetail
)

type action int

const (
	actionLogin action = iota
	actionRegister
	actionConfirm
	actionResend
	actionForgot
	actionReset
	actionLogout
)

type menuItem struct {
	id, title, desc string
}

type formData struct {
	email, password, password2, code string
}

type doneMsg struct {
	kind     action
	code     string
	user     apiclient.User
	products []market.Product
	cart     market.Cart
	orders   []market.Order
	order    market.Order
	err      error
}

type model struct {
	api       *apiclient.Client
	data      *formData
	screen    screen
	width     int
	height    int
	items     []menuItem
	cursor    int
	form      *huh.Form
	formTitle string
	pending   action
	spinner   spinner.Model
	status    string
	statusOK  bool
	authed    bool
	session   apiclient.User

	pendingEmail string
	pendingCode  string
	noticeLead   string
	noticeNext   action
	loadingText  string
	help         viewport.Model

	section     section
	shop        *shopFields
	products    []market.Product
	product     market.Product
	cart        market.Cart
	orders      []market.Order
	order       market.Order
	lineRef     string
	detail      viewport.Model
	detailTitle string
}

// NewModel construit l'écran d'accueil. apiURL est l'adresse du serveur HTTP.
func NewModel(apiURL string) model {
	m := model{
		api:     apiclient.New(apiURL),
		data:    &formData{},
		spinner: frame.Client.NewSpinner(),
		help:    viewport.New(72, 16),
		shop:    &shopFields{},
		detail:  viewport.New(72, 16),
		width:   80,
		height:  24,
		screen:  screenMenu,
	}
	m.useGuestMenu()
	m.renderHelp()
	return m
}

// Run lance l'interface dans le terminal courant.
func Run(apiURL string) error {
	program := tea.NewProgram(NewModel(apiURL), tea.WithAltScreen(), tea.WithReportFocus())
	_, err := program.Run()
	return err
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if done, ok := msg.(doneMsg); ok {
		return m.onDone(done)
	}
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = size.Width
		m.height = size.Height
		m.renderHelp()
		m.detail.Width = m.contentWidth()
		m.detail.Height = m.helpHeight()
	}

	switch m.screen {
	case screenLoading:
		if _, ok := msg.(spinner.TickMsg); ok {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	case screenHelp:
		if key, ok := msg.(tea.KeyMsg); ok && (key.String() == "enter" || key.String() == "esc") {
			m.screen = screenMenu
			return m, nil
		}
		var cmd tea.Cmd
		m.help, cmd = m.help.Update(msg)
		return m, cmd
	case screenDetail:
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "enter", "esc", "q":
				return m.goBack()
			}
		}
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd
	case screenNotice:
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "enter":
				return m.openForm(m.noticeNext, true)
			case "esc":
				m.screen = screenMenu
			}
		}
		return m, nil
	case screenForm:
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
			return m.leaveForm()
		}
		next, cmd := m.form.Update(msg)
		form, ok := next.(*huh.Form)
		if !ok {
			return m, cmd
		}
		m.form = form
		switch m.form.State {
		case huh.StateCompleted:
			m.screen = screenLoading
			m.loadingText = loadingText(m.pending)
			return m, tea.Batch(m.spinner.Tick, m.submit())
		case huh.StateAborted:
			return m.leaveForm()
		}
		return m, cmd
	default:
		return m.updateMenu(msg)
	}
}

func (m model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "enter":
		item, ok := m.current()
		if !ok {
			return m, nil
		}
		return m.activate(item.id)
	case "q":
		if m.authed && m.section != sectionHome {
			return m.goBack()
		}
		return m, tea.Quit
	case "esc":
		if m.authed && m.section != sectionHome {
			return m.goBack()
		}
	case "up", "k":
		m.move(-m.columns())
	case "down", "j":
		m.move(m.columns())
	case "left", "h":
		m.move(-1)
	case "right", "l":
		m.move(1)
	default:
		if n, ok := menuDigit(key.String()); ok {
			m.jump(n)
		}
	}
	return m, nil
}

func (m model) View() string {
	var body string
	switch m.screen {
	case screenForm:
		parts := []string{m.form.View()}
		if intro := m.formIntro(); intro != "" {
			parts = []string{intro, m.form.View()}
		}
		body = lipgloss.JoinVertical(lipgloss.Left, parts...)
	case screenNotice:
		body = m.noticeBody()
	case screenHelp:
		body = m.help.View()
	case screenDetail:
		body = m.detail.View()
	case screenLoading:
		body = m.spinner.View() + "  " + frame.Client.TextLine(m.loadingText)
	default:
		body = m.gallery()
	}
	return frame.Client.Screen(m.width, m.height, m.heading(), m.tagline(), m.status, m.statusOK, body, m.hint())
}

func (m model) heading() string {
	switch m.screen {
	case screenForm:
		return m.formTitle
	case screenNotice:
		return "Code de confirmation"
	case screenHelp:
		return "Aide"
	case screenLoading:
		return "Veuillez patienter"
	default:
		if title, ok := m.shopHeading(); ok {
			return title
		}
		if m.authed && m.session.Email != "" {
			return m.session.Email
		}
		return "Bienvenue"
	}
}

func (m model) tagline() string {
	if text, ok := m.shopTagline(); ok && (m.screen == screenMenu || m.screen == screenDetail) {
		return text
	}
	if m.screen != screenMenu {
		return ""
	}
	if m.authed {
		return "Session " + m.session.Role
	}
	return "Plateforme d'achat"
}

func (m *model) activate(id string) (tea.Model, tea.Cmd) {
	switch id {
	case "help":
		m.screen = screenHelp
		m.renderHelp()
		return *m, nil
	case "quit":
		return *m, tea.Quit
	case "logout":
		m.status = ""
		m.pending = actionLogout
		m.screen = screenLoading
		m.loadingText = loadingText(actionLogout)
		return *m, tea.Batch(m.spinner.Tick, m.submit())
	case "login":
		m.status = ""
		return m.openForm(actionLogin, true)
	case "register":
		m.status = ""
		return m.openForm(actionRegister, true)
	case "confirm":
		m.status = ""
		return m.openForm(actionConfirm, true)
	case "resend":
		m.status = ""
		return m.openForm(actionResend, true)
	case "forgot":
		m.status = ""
		return m.openForm(actionForgot, true)
	default:
		return m.activateShop(id)
	}
}

func (m *model) openForm(kind action, reset bool) (tea.Model, tea.Cmd) {
	if reset {
		m.data.email = m.pendingEmail
		m.data.password = ""
		m.data.password2 = ""
		m.data.code = ""
	}
	m.pending = kind
	m.formTitle = titleFor(kind)
	m.form = m.buildForm(kind)
	m.screen = screenForm
	return *m, m.form.Init()
}

func (m *model) onDone(msg doneMsg) (tea.Model, tea.Cmd) {
	if isShop(msg.kind) {
		return m.finishShop(msg)
	}
	if msg.err != nil {
		m.status = msg.err.Error()
		m.statusOK = false
		if msg.kind == actionLogout {
			m.signOut()
			return *m, nil
		}
		return m.openForm(msg.kind, false)
	}

	switch msg.kind {
	case actionRegister:
		m.rememberEmail()
		m.pendingCode = msg.code
		m.noticeLead = "Compte créé. Confirmez-le avec ce code : il remplace l'email de confirmation."
		m.noticeNext = actionConfirm
		m.screen = screenNotice
		m.status = ""
		return *m, nil
	case actionResend:
		if msg.code == "" {
			m.status = "Aucun compte en attente pour cette adresse."
			m.statusOK = false
			m.screen = screenMenu
			return *m, nil
		}
		m.rememberEmail()
		m.pendingCode = msg.code
		m.noticeLead = "Nouveau code de confirmation."
		m.noticeNext = actionConfirm
		m.screen = screenNotice
		m.status = ""
		return *m, nil
	case actionConfirm:
		m.status = "Compte confirmé. Vous pouvez vous connecter."
		m.statusOK = true
		m.pendingCode = ""
		return m.openForm(actionLogin, true)
	case actionLogin:
		m.session = msg.user
		m.authed = true
		m.status = "Connexion réussie"
		m.statusOK = true
		m.pendingCode = ""
		m.screen = screenMenu
		m.useSessionMenu()
		return *m, nil
	case actionForgot:
		if msg.code == "" {
			m.status = "Si un compte existe, un code a été généré. Rien n'a été renvoyé pour cette adresse."
			m.statusOK = false
			m.screen = screenMenu
			return *m, nil
		}
		m.rememberEmail()
		m.pendingCode = msg.code
		m.noticeLead = "Code de réinitialisation. Choisissez ensuite un nouveau mot de passe."
		m.noticeNext = actionReset
		m.screen = screenNotice
		m.status = ""
		return *m, nil
	case actionReset:
		m.status = "Mot de passe mis à jour. Connectez-vous avec le nouveau."
		m.statusOK = true
		m.pendingCode = ""
		return m.openForm(actionLogin, true)
	case actionLogout:
		m.signOut()
		m.status = "Session fermée"
		m.statusOK = true
		return *m, nil
	default:
		m.screen = screenMenu
		return *m, nil
	}
}

func (m *model) submit() tea.Cmd {
	if isShop(m.pending) {
		return m.submitShop()
	}
	kind := m.pending
	email := m.data.email
	password := m.data.password
	code := m.data.code
	api := m.api
	return func() tea.Msg {
		ctx := context.Background()
		switch kind {
		case actionLogin:
			user, err := api.Login(ctx, email, password)
			return doneMsg{kind: kind, user: user, err: err}
		case actionRegister:
			confirmation, err := api.Register(ctx, email, password)
			return doneMsg{kind: kind, code: confirmation, err: err}
		case actionConfirm:
			return doneMsg{kind: kind, err: api.Confirm(ctx, email, code)}
		case actionResend:
			confirmation, err := api.RenewConfirmation(ctx, email)
			return doneMsg{kind: kind, code: confirmation, err: err}
		case actionForgot:
			reset, err := api.ForgotPassword(ctx, email)
			return doneMsg{kind: kind, code: reset, err: err}
		case actionReset:
			return doneMsg{kind: kind, err: api.ResetPassword(ctx, email, code, password)}
		case actionLogout:
			return doneMsg{kind: kind, err: api.Logout(ctx)}
		default:
			return doneMsg{kind: kind, err: errors.New("action inconnue")}
		}
	}
}

func (m *model) signOut() {
	m.authed = false
	m.session = apiclient.User{}
	m.api.SetToken("")
	m.section = sectionHome
	m.products = nil
	m.product = market.Product{}
	m.cart = market.Cart{}
	m.orders = nil
	m.order = market.Order{}
	m.lineRef = ""
	m.useGuestMenu()
	m.screen = screenMenu
}

func (m *model) rememberEmail() {
	m.pendingEmail = m.data.email
}

func (m *model) useGuestMenu() {
	m.setMenu([]menuItem{
		{id: "login", title: "Se connecter", desc: "Email et mot de passe"},
		{id: "register", title: "Créer un compte", desc: "Inscription, puis un code"},
		{id: "confirm", title: "Confirmer", desc: "Saisir le code d'inscription"},
		{id: "resend", title: "Renvoyer le code", desc: "Si le précédent a expiré"},
		{id: "forgot", title: "Mot de passe oublié", desc: "Réinitialiser sans connexion"},
		{id: "help", title: "Aide", desc: "Parcours d'authentification"},
		{id: "quit", title: "Quitter", desc: "Fermer l'application"},
	})
}

func (m *model) useSessionMenu() {
	m.section = sectionHome
	m.setMenu([]menuItem{
		{id: "search", title: "Rechercher", desc: "Nom, catégorie, prix HT ou TTC"},
		{id: "cart", title: "Mon panier", desc: "Quantités, total TTC, livraison offerte"},
		{id: "orders", title: "Mes commandes", desc: "Date, produits et statut"},
		{id: "help", title: "Aide", desc: "Raccourcis et rappels"},
		{id: "logout", title: "Se déconnecter", desc: "Ferme la session"},
		{id: "quit", title: "Quitter", desc: "Fermer l'application"},
	})
}

func (m *model) renderHelp() {
	m.help.Width = m.contentWidth()
	m.help.Height = m.helpHeight()
	m.help.SetContent(renderMarkdown(helpMarkdown, m.contentWidth()))
}

func (m model) helpHeight() int {
	used := 6
	if m.status != "" {
		used += 2
	}
	height := m.height - used
	if height < 5 {
		height = 5
	}
	return height
}

func (m model) contentWidth() int {
	width := frame.Client.PanelWidth(m.width) - 2
	if width < 24 {
		width = 24
	}
	return width
}

func (m model) hint() string {
	switch m.screen {
	case screenForm:
		return frame.Client.Keys("entrée", "valider", "maj+tab", "précédent", "esc", "retour", "ctrl+c", "quitter")
	case screenDetail:
		return frame.Client.Keys("↑↓", "défiler", "esc", "retour", "q", "retour", "ctrl+c", "quitter")
	case screenHelp:
		return frame.Client.Keys("↑↓", "défiler", "entrée", "menu", "esc", "menu", "ctrl+c", "quitter")
	case screenNotice:
		return frame.Client.Keys("entrée", "continuer", "esc", "menu", "ctrl+c", "quitter")
	case screenLoading:
		return frame.Client.Keys("ctrl+c", "quitter")
	default:
		if m.authed && m.section != sectionHome {
			return frame.Client.Keys("flèches", "naviguer", "entrée", "ouvrir", "esc", "retour", "q", "retour")
		}
		return frame.Client.Keys("flèches", "naviguer", "1-9", "choisir", "entrée", "ouvrir", "q", "quitter")
	}
}

func (m model) formIntro() string {
	if intro := m.shopIntro(); intro != "" {
		return frame.Client.Faint(intro) + "\n\n"
	}
	if m.pendingCode == "" || (m.pending != actionConfirm && m.pending != actionReset) {
		return ""
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		frame.Client.Faint("Code à saisir"),
		frame.Client.Code(m.pendingCode),
		"",
	)
}

func (m model) noticeBody() string {
	card := frame.Client.Card(m.noticeLead, frame.Client.PanelWidth(m.width))
	return lipgloss.JoinVertical(lipgloss.Left,
		card,
		"",
		frame.Client.Faint("Code"),
		frame.Client.Code(m.pendingCode),
	)
}

func (m *model) setMenu(items []menuItem) {
	m.items = items
	m.cursor = 0
}

func (m model) current() (menuItem, bool) {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return menuItem{}, false
	}
	return m.items[m.cursor], true
}

func (m model) panel() int {
	return frame.Client.PanelWidth(m.width)
}

func (m model) galleryHeight() int {
	used := 7
	if m.tagline() != "" {
		used++
	}
	if m.status != "" {
		used += 2
	}
	height := m.height - used
	if height < cardFloor {
		height = cardFloor
	}
	return height
}

const cardFloor = 5

func (m model) columns() int {
	cols, _ := frame.Client.GridShape(len(m.items), m.panel(), m.galleryHeight())
	if cols < 1 {
		return 1
	}
	return cols
}

func (m *model) move(delta int) {
	if len(m.items) == 0 || delta == 0 {
		return
	}
	next := m.cursor + delta
	if next < 0 {
		next = 0
	}
	if next >= len(m.items) {
		next = len(m.items) - 1
	}
	m.cursor = next
}

func (m *model) jump(local int) {
	_, pageSize := frame.Client.GridShape(len(m.items), m.panel(), m.galleryHeight())
	if pageSize < 1 || local < 0 {
		return
	}
	page := 0
	if m.cursor >= 0 {
		page = m.cursor / pageSize
	}
	index := page*pageSize + local
	if index >= 0 && index < len(m.items) {
		m.cursor = index
	}
}

func (m model) gallery() string {
	choices := make([]frame.Choice, len(m.items))
	for i, item := range m.items {
		choices[i] = frame.Choice{Title: item.title, Detail: item.desc}
	}
	return frame.Client.Gallery(choices, m.cursor, m.panel(), m.galleryHeight())
}

func menuDigit(key string) (int, bool) {
	if len(key) != 1 || key[0] < '1' || key[0] > '9' {
		return 0, false
	}
	return int(key[0] - '1'), true
}

func loadingText(kind action) string {
	switch kind {
	case actionLogin:
		return "Connexion..."
	case actionRegister:
		return "Création du compte..."
	case actionConfirm:
		return "Confirmation..."
	case actionResend:
		return "Nouveau code..."
	case actionForgot:
		return "Demande de code..."
	case actionReset:
		return "Mise à jour du mot de passe..."
	case actionLogout:
		return "Fermeture de la session..."
	default:
		return "Échange avec le serveur..."
	}
}
