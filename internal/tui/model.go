package tui

import (
	"context"
	"errors"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/madoxSio/go-cli-market/internal/apiclient"
)

type screen int

const (
	screenMenu screen = iota
	screenForm
	screenNotice
	screenHelp
	screenLoading
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

func (item menuItem) Title() string       { return item.title }
func (item menuItem) Description() string { return item.desc }
func (item menuItem) FilterValue() string { return item.title }

type formData struct {
	email, password, password2, code string
}

type doneMsg struct {
	kind action
	code string
	user apiclient.User
	err  error
}

type model struct {
	api       *apiclient.Client
	data      *formData
	screen    screen
	width     int
	height    int
	list      list.Model
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
	helpView     string
}

// NewModel construit l'écran d'accueil. apiURL est l'adresse du serveur HTTP.
func NewModel(apiURL string) model {
	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#C4B5FD"))

	m := model{
		api:     apiclient.New(apiURL),
		data:    &formData{},
		spinner: spin,
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
		m.syncListSize()
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
		}
		return m, nil
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
			m.screen = screenMenu
			return m, nil
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
			m.screen = screenMenu
			return m, nil
		}
		return m, cmd
	default:
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "enter":
				item, ok := m.list.SelectedItem().(menuItem)
				if !ok {
					return m, nil
				}
				return m.activate(item.id)
			case "q":
				return m, tea.Quit
			}
		}
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}
}

func (m model) View() string {
	var body string
	switch m.screen {
	case screenForm:
		body = lipgloss.JoinVertical(lipgloss.Left, formTitleStyle.Render(m.formTitle), m.formIntro(), m.form.View())
	case screenNotice:
		body = m.noticeBody()
	case screenHelp:
		body = m.helpView
	case screenLoading:
		body = m.spinner.View() + "  " + m.loadingText
	default:
		if m.authed {
			body = m.accountCard() + "\n" + m.list.View()
		} else {
			body = m.list.View()
		}
	}
	return m.layout(body)
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
		return *m, nil
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
	m.useGuestMenu()
	m.screen = screenMenu
}

func (m *model) rememberEmail() {
	m.pendingEmail = m.data.email
}

func (m *model) useGuestMenu() {
	m.list = newMenu("Menu", []menuItem{
		{id: "login", title: "Se connecter", desc: "Email et mot de passe"},
		{id: "register", title: "Créer un compte", desc: "Inscription, puis code de confirmation"},
		{id: "confirm", title: "Confirmer l'inscription", desc: "Saisir le code affiché à l'inscription"},
		{id: "resend", title: "Renvoyer le code", desc: "Nouveau code si le précédent a expiré"},
		{id: "forgot", title: "Mot de passe oublié", desc: "Réinitialiser sans être connecté"},
		{id: "help", title: "Aide", desc: "Parcours d'authentification"},
		{id: "quit", title: "Quitter", desc: "Fermer l'application"},
	})
	m.syncListSize()
}

func (m *model) useSessionMenu() {
	m.list = newMenu("Mon compte", []menuItem{
		{id: "help", title: "Aide", desc: "Raccourcis et rappels"},
		{id: "logout", title: "Se déconnecter", desc: "Ferme la session sur le serveur"},
		{id: "quit", title: "Quitter", desc: "Fermer l'application"},
	})
	m.syncListSize()
}

func (m *model) renderHelp() {
	m.helpView = renderMarkdown(helpMarkdown, m.contentWidth())
}

func (m *model) syncListSize() {
	width := m.width
	if width < 20 {
		width = 20
	}
	height := m.height - 10
	if m.authed {
		height -= 6
	}
	if height < 8 {
		height = 8
	}
	m.list.SetSize(width, height)
}

func (m model) contentWidth() int {
	width := m.width - 2
	if width < 24 {
		width = 24
	}
	if width > 88 {
		width = 88
	}
	return width
}

func (m model) layout(body string) string {
	parts := []string{
		titleStyle.Render("go-cli-market"),
		subtitleStyle.Render("Plateforme d'achat — client"),
	}
	if m.status != "" {
		if m.statusOK {
			parts = append(parts, okStyle.Render(m.status))
		} else {
			parts = append(parts, errStyle.Render(m.status))
		}
	}
	parts = append(parts, body, hintStyle.Render(m.hint()))
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m model) hint() string {
	switch m.screen {
	case screenForm:
		return "entrée valider   maj+tab champ précédent   esc menu   ctrl+c quitter"
	case screenHelp, screenNotice:
		return "entrée continuer   esc menu   ctrl+c quitter"
	case screenLoading:
		return "ctrl+c quitter"
	default:
		return "↑↓ choisir   entrée valider   q quitter"
	}
}

func (m model) formIntro() string {
	if m.pendingCode == "" || (m.pending != actionConfirm && m.pending != actionReset) {
		return ""
	}
	return lipgloss.JoinVertical(lipgloss.Left, "Code à saisir", codeStyle.Render(m.pendingCode), "")
}

func (m model) noticeBody() string {
	lines := []string{m.noticeLead, "", "Code", codeStyle.Render(m.pendingCode)}
	return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m model) accountCard() string {
	content := lipgloss.JoinVertical(lipgloss.Left,
		okStyle.Render("Session ouverte"),
		"Email   "+m.session.Email,
		"Rôle    "+m.session.Role,
		"",
		"Le catalogue, le panier et les commandes seront branchés ensuite.",
	)
	return boxStyle.Render(content)
}

func newMenu(title string, items []menuItem) list.Model {
	rows := make([]list.Item, len(items))
	for i, item := range items {
		rows[i] = item
	}
	menu := list.New(rows, list.NewDefaultDelegate(), 0, 0)
	menu.Title = title
	menu.SetFilteringEnabled(false)
	menu.SetShowStatusBar(false)
	menu.DisableQuitKeybindings()
	menu.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("#6D28D9")).
		Foreground(lipgloss.Color("#FAFAFA")).
		Padding(0, 1)
	return menu
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
