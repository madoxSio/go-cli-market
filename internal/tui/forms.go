package tui

import (
	"errors"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/madoxSio/go-cli-market/internal/frame"
)

func titleFor(kind action) string {
	switch kind {
	case actionLogin:
		return "Connexion"
	case actionRegister:
		return "Créer un compte"
	case actionConfirm:
		return "Confirmer l'inscription"
	case actionResend:
		return "Renvoyer le code"
	case actionForgot:
		return "Mot de passe oublié"
	case actionReset:
		return "Nouveau mot de passe"
	default:
		return ""
	}
}

func (m *model) buildForm(kind action) *huh.Form {
	email := huh.NewInput().
		Title("Email").
		Placeholder("ada@example.com").
		CharLimit(254).
		Value(&m.data.email).
		Validate(validateEmailField)

	switch kind {
	case actionLogin:
		return m.finishForm(huh.NewGroup(
			email,
			huh.NewInput().
				Title("Mot de passe").
				EchoMode(huh.EchoModePassword).
				CharLimit(72).
				Value(&m.data.password).
				Validate(requireText("mot de passe requis")),
		))
	case actionRegister:
		return m.finishForm(huh.NewGroup(
			email.Description("Un code de confirmation sera affiché."),
			huh.NewInput().
				Title("Mot de passe").
				Description("8 caractères minimum.").
				EchoMode(huh.EchoModePassword).
				CharLimit(72).
				Value(&m.data.password).
				Validate(validatePasswordField),
			huh.NewInput().
				Title("Confirmation").
				EchoMode(huh.EchoModePassword).
				CharLimit(72).
				Value(&m.data.password2).
				Validate(m.samePassword),
		))
	case actionConfirm:
		return m.finishForm(huh.NewGroup(
			email,
			huh.NewInput().
				Title("Code").
				Description("La casse n'est pas prise en compte.").
				CharLimit(16).
				Value(&m.data.code).
				Validate(requireText("code requis")),
		))
	case actionResend, actionForgot:
		return m.finishForm(huh.NewGroup(email))
	case actionReset:
		return m.finishForm(huh.NewGroup(
			email,
			huh.NewInput().
				Title("Code").
				CharLimit(16).
				Value(&m.data.code).
				Validate(requireText("code requis")),
			huh.NewInput().
				Title("Nouveau mot de passe").
				EchoMode(huh.EchoModePassword).
				CharLimit(72).
				Value(&m.data.password).
				Validate(validatePasswordField),
			huh.NewInput().
				Title("Confirmation").
				EchoMode(huh.EchoModePassword).
				CharLimit(72).
				Value(&m.data.password2).
				Validate(m.samePassword),
		))
	default:
		return m.finishForm(huh.NewGroup(email))
	}
}

func (m *model) finishForm(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).
		WithTheme(frame.Client.FormTheme()).
		WithShowHelp(false).
		WithWidth(m.contentWidth())
}

func (m *model) samePassword(value string) error {
	if value != m.data.password {
		return errors.New("les mots de passe diffèrent")
	}
	return nil
}

func validateEmailField(value string) error {
	value = strings.TrimSpace(value)
	if strings.Count(value, "@") != 1 {
		return errors.New("email invalide")
	}
	local, domain, _ := strings.Cut(value, "@")
	if local == "" || !strings.Contains(domain, ".") {
		return errors.New("email invalide")
	}
	return nil
}

func validatePasswordField(value string) error {
	if len(value) < 8 {
		return errors.New("8 caractères minimum")
	}
	return nil
}

func requireText(message string) func(string) error {
	return func(value string) error {
		if strings.TrimSpace(value) == "" {
			return errors.New(message)
		}
		return nil
	}
}
