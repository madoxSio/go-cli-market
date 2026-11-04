package admintui

import (
	"context"
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/madoxSio/go-cli-market/internal/apiclient"
	"github.com/madoxSio/go-cli-market/internal/frame"
)

type screen int

const (
	screenLogin screen = iota
	screenMenu
	screenUsers
	screenUser
	screenForm
	screenHelp
	screenLoading
)

type action int

const (
	actLogin action = iota
	actList
	actCreate
	actUpdate
	actDelete
	actConfirm
	actLogout
)

type menuItem struct {
	id, title, desc string
}

type formData struct {
	email, password, role string
	confirmed             bool
}

type doneMsg struct {
	kind  action
	user  apiclient.User
	users []apiclient.User
	err   error
}

type model struct {
	api      *apiclient.Client
	data     *formData
	screen   screen
	back     screen
	width    int
	height   int
	items    []menuItem
	cursor   int
	form     *huh.Form
	title    string
	pending  action
	spinner  spinner.Model
	status   string
	statusOK bool
	authed   bool
	session  apiclient.User
	users    []apiclient.User
	selected apiclient.User
	help     viewport.Model
	loading  string
}

// NewModel construit l'écran d'administration.
func NewModel(apiURL string) model {
	m := model{
		api:     apiclient.New(apiURL),
		data:    &formData{role: "client"},
		spinner: frame.Admin.NewSpinner(),
		help:    viewport.New(72, 16),
		width:   80,
		height:  24,
		screen:  screenLogin,
	}
	m.openLogin()
	m.renderHelp()
	return m
}

// Run lance l'administration dans le terminal courant.
func Run(apiURL string) error {
	program := tea.NewProgram(NewModel(apiURL), tea.WithAltScreen(), tea.WithReportFocus())
	_, err := program.Run()
	return err
}

func (m model) Init() tea.Cmd { return m.form.Init() }

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
	case screenLogin:
		return m.updateForm(msg)
	case screenForm:
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
			m.screen = m.back
			if m.screen == screenUsers {
				m.useUserList()
			}
			if m.screen == screenUser {
				m.useUserMenu()
			}
			return m, nil
		}
		return m.updateForm(msg)
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
		if m.screen == screenMenu || m.screen == screenLogin {
			return m, tea.Quit
		}
	case "esc":
		if m.screen == screenUsers {
			m.screen = screenMenu
			m.useMainMenu()
			return m, nil
		}
		if m.screen == screenUser {
			return m.showUsers()
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
	case screenLogin, screenForm:
		body = m.form.View()
	case screenHelp:
		body = m.help.View()
	case screenLoading:
		body = m.spinner.View() + "  " + frame.Admin.TextLine(m.loading)
	default:
		body = m.gallery()
	}
	return frame.Admin.Screen(m.width, m.height, m.heading(), m.tagline(), m.status, m.statusOK, body, m.hint())
}

func (m model) heading() string {
	switch m.screen {
	case screenLogin, screenForm:
		return m.title
	case screenHelp:
		return "Aide"
	case screenLoading:
		return "Veuillez patienter"
	case screenUsers:
		return "Comptes"
	case screenUser:
		if m.selected.Email != "" {
			return m.selected.Email
		}
		return "Compte"
	default:
		if m.authed && m.session.Email != "" {
			return m.session.Email
		}
		return "Tableau de bord"
	}
}

func (m model) tagline() string {
	switch m.screen {
	case screenLogin:
		return "Réservé aux administrateurs"
	case screenMenu:
		return "Gestion des comptes"
	case screenUser:
		if m.selected.Role == "" {
			return ""
		}
		state := "en attente de confirmation"
		if m.selected.Confirmed {
			state = "confirmé"
		}
		return m.selected.Role + " · " + state
	default:
		return ""
	}
}

func (m *model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.form.Update(msg)
	form, ok := next.(*huh.Form)
	if !ok {
		return *m, cmd
	}
	m.form = form
	switch m.form.State {
	case huh.StateCompleted:
		if m.pending == actDelete && !m.data.confirmed {
			m.screen = screenUser
			m.useUserMenu()
			return *m, nil
		}
		m.screen = screenLoading
		m.loading = loadingText(m.pending)
		return *m, tea.Batch(m.spinner.Tick, m.submit())
	case huh.StateAborted:
		if m.pending == actLogin {
			m.openLogin()
			return *m, m.form.Init()
		}
		m.screen = m.back
		return *m, nil
	}
	return *m, cmd
}

func (m *model) activate(id string) (tea.Model, tea.Cmd) {
	switch id {
	case "users":
		return m.loadUsers()
	case "help":
		m.screen = screenHelp
		m.renderHelp()
		return *m, nil
	case "logout":
		m.pending = actLogout
		m.screen = screenLoading
		m.loading = loadingText(actLogout)
		return *m, tea.Batch(m.spinner.Tick, m.submit())
	case "quit":
		return *m, tea.Quit
	case "create":
		m.data.email = ""
		m.data.password = ""
		m.data.role = "client"
		m.data.confirmed = true
		return m.showForm(actCreate, "Créer un compte", screenUsers, m.createForm())
	case "edit":
		m.data.email = m.selected.Email
		m.data.password = ""
		m.data.role = m.selected.Role
		return m.showForm(actUpdate, "Modifier "+m.selected.Email, screenUser, m.editForm())
	case "confirm":
		m.pending = actConfirm
		m.screen = screenLoading
		m.loading = loadingText(actConfirm)
		return *m, tea.Batch(m.spinner.Tick, m.submit())
	case "delete":
		m.data.confirmed = false
		return m.showForm(actDelete, "Suppression", screenUser, m.deleteForm())
	case "back":
		return m.showUsers()
	default:
		if len(id) > 5 && id[:5] == "user:" {
			userID, err := strconv.ParseInt(id[5:], 10, 64)
			if err != nil {
				return *m, nil
			}
			for _, user := range m.users {
				if user.ID == userID {
					m.selected = user
					m.screen = screenUser
					m.useUserMenu()
					return *m, nil
				}
			}
		}
		return *m, nil
	}
}

func (m *model) showForm(kind action, title string, back screen, form *huh.Form) (tea.Model, tea.Cmd) {
	m.pending = kind
	m.title = title
	m.back = back
	m.form = form
	m.screen = screenForm
	m.status = ""
	return *m, m.form.Init()
}

func (m *model) loadUsers() (tea.Model, tea.Cmd) {
	m.pending = actList
	m.back = screenMenu
	m.screen = screenLoading
	m.loading = loadingText(actList)
	return *m, tea.Batch(m.spinner.Tick, m.submit())
}

func (m *model) showUsers() (tea.Model, tea.Cmd) {
	m.screen = screenUsers
	m.useUserList()
	return *m, nil
}

func (m *model) onDone(msg doneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status = msg.err.Error()
		m.statusOK = false
		if msg.kind == actLogin {
			m.openLogin()
			return *m, m.form.Init()
		}
		if msg.kind == actLogout {
			m.signOut()
			return *m, m.form.Init()
		}
		m.screen = m.back
		switch m.screen {
		case screenUsers:
			m.useUserList()
		case screenUser:
			m.useUserMenu()
		case screenMenu:
			m.useMainMenu()
		default:
			m.openLogin()
			return *m, m.form.Init()
		}
		return *m, nil
	}

	switch msg.kind {
	case actLogin:
		if msg.user.Role != "admin" {
			m.api.SetToken("")
			m.status = "Ce compte n'est pas administrateur."
			m.statusOK = false
			m.openLogin()
			return *m, m.form.Init()
		}
		m.session = msg.user
		m.authed = true
		m.status = "Connexion administrateur"
		m.statusOK = true
		m.screen = screenMenu
		m.useMainMenu()
		return *m, nil
	case actList:
		m.users = msg.users
		m.statusOK = true
		return m.showUsers()
	case actCreate, actUpdate, actConfirm, actDelete:
		m.status = successText(msg.kind)
		m.statusOK = true
		return m.loadUsers()
	case actLogout:
		m.signOut()
		m.status = "Session fermée"
		m.statusOK = true
		return *m, m.form.Init()
	default:
		m.screen = screenMenu
		return *m, nil
	}
}

func (m *model) submit() tea.Cmd {
	kind := m.pending
	data := *m.data
	selected := m.selected
	api := m.api
	return func() tea.Msg {
		ctx := context.Background()
		switch kind {
		case actLogin:
			user, err := api.Login(ctx, data.email, data.password)
			return doneMsg{kind: kind, user: user, err: err}
		case actList:
			users, err := api.ListUsers(ctx)
			return doneMsg{kind: kind, users: users, err: err}
		case actCreate:
			user, err := api.CreateUser(ctx, data.email, data.password, data.role, data.confirmed)
			return doneMsg{kind: kind, user: user, err: err}
		case actUpdate:
			user, err := api.UpdateUser(ctx, selected.ID, data.email, data.password, data.role)
			return doneMsg{kind: kind, user: user, err: err}
		case actConfirm:
			err := api.ConfirmUser(ctx, selected.ID)
			return doneMsg{kind: kind, err: err}
		case actDelete:
			err := api.DeleteUser(ctx, selected.ID)
			return doneMsg{kind: kind, err: err}
		case actLogout:
			return doneMsg{kind: kind, err: api.Logout(ctx)}
		default:
			return doneMsg{kind: kind}
		}
	}
}

func (m *model) openLogin() {
	m.screen = screenLogin
	m.pending = actLogin
	m.back = screenLogin
	m.title = "Connexion"
	m.data.password = ""
	m.form = m.loginForm()
	m.items = nil
	m.cursor = 0
}

func (m *model) signOut() {
	m.authed = false
	m.session = apiclient.User{}
	m.users = nil
	m.api.SetToken("")
	m.openLogin()
}

func (m *model) useMainMenu() {
	m.setMenu([]menuItem{
		{id: "users", title: "Comptes", desc: "Créer, modifier, confirmer"},
		{id: "help", title: "Aide", desc: "Rôle de cette application"},
		{id: "logout", title: "Se déconnecter", desc: "Ferme la session"},
		{id: "quit", title: "Quitter", desc: "Fermer l'application"},
	})
}

func (m *model) useUserList() {
	items := []menuItem{{id: "create", title: "Créer un compte", desc: "Client ou administrateur"}}
	for _, user := range m.users {
		state := "en attente"
		if user.Confirmed {
			state = "confirmé"
		}
		items = append(items, menuItem{
			id:    fmt.Sprintf("user:%d", user.ID),
			title: user.Email,
			desc:  user.Role + " · " + state,
		})
	}
	m.setMenu(items)
	m.screen = screenUsers
}

func (m *model) useUserMenu() {
	items := []menuItem{{id: "edit", title: "Modifier", desc: "Email, rôle, mot de passe"}}
	if !m.selected.Confirmed {
		items = append(items, menuItem{id: "confirm", title: "Confirmer", desc: "Sans code d'inscription"})
	}
	items = append(items,
		menuItem{id: "delete", title: "Supprimer", desc: m.selected.Email},
		menuItem{id: "back", title: "Retour", desc: "Liste des comptes"},
	)
	m.setMenu(items)
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
	width := frame.Admin.PanelWidth(m.width) - 2
	if width < 24 {
		width = 24
	}
	return width
}

func (m model) hint() string {
	switch m.screen {
	case screenForm:
		return frame.Admin.Keys("entrée", "valider", "maj+tab", "précédent", "esc", "retour", "ctrl+c", "quitter")
	case screenHelp:
		return frame.Admin.Keys("↑↓", "défiler", "entrée", "retour", "esc", "retour", "ctrl+c", "quitter")
	case screenLoading:
		return frame.Admin.Keys("ctrl+c", "quitter")
	case screenLogin:
		return frame.Admin.Keys("entrée", "valider", "ctrl+c", "quitter")
	default:
		return frame.Admin.Keys("flèches", "naviguer", "1-9", "choisir", "entrée", "ouvrir", "esc", "retour", "q", "quitter")
	}
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
	return frame.Admin.PanelWidth(m.width)
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
	if height < 5 {
		height = 5
	}
	return height
}

func (m model) columns() int {
	cols, _ := frame.Admin.GridShape(len(m.items), m.panel(), m.galleryHeight())
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
	_, pageSize := frame.Admin.GridShape(len(m.items), m.panel(), m.galleryHeight())
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
	return frame.Admin.Gallery(choices, m.cursor, m.panel(), m.galleryHeight())
}

func menuDigit(key string) (int, bool) {
	if len(key) != 1 || key[0] < '1' || key[0] > '9' {
		return 0, false
	}
	return int(key[0] - '1'), true
}

func loadingText(kind action) string {
	switch kind {
	case actLogin:
		return "Connexion..."
	case actList:
		return "Chargement des comptes..."
	case actCreate:
		return "Création..."
	case actUpdate:
		return "Mise à jour..."
	case actConfirm:
		return "Confirmation..."
	case actDelete:
		return "Suppression..."
	case actLogout:
		return "Fermeture de la session..."
	default:
		return "Échange avec le serveur..."
	}
}

func successText(kind action) string {
	switch kind {
	case actCreate:
		return "Compte créé"
	case actUpdate:
		return "Compte mis à jour"
	case actConfirm:
		return "Compte confirmé"
	case actDelete:
		return "Compte supprimé"
	default:
		return "Opération réussie"
	}
}
