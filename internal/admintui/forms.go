package admintui

import (
	"errors"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/madoxSio/go-cli-market/internal/frame"
)

func (m *model) loginForm() *huh.Form {
	return m.finish(huh.NewGroup(
		huh.NewInput().
			Title("Email").
			Description("Compte administrateur.").
			Placeholder("admin@market.local").
			CharLimit(254).
			Value(&m.data.email).
			Validate(validateEmailField),
		huh.NewInput().
			Title("Mot de passe").
			EchoMode(huh.EchoModePassword).
			CharLimit(72).
			Value(&m.data.password).
			Validate(func(value string) error {
				if strings.TrimSpace(value) == "" {
					return errors.New("mot de passe requis")
				}
				return nil
			}),
	))
}

func (m *model) createForm() *huh.Form {
	roles := huh.NewOptions("client", "admin")
	return m.finish(huh.NewGroup(
		huh.NewInput().Title("Email").CharLimit(254).Value(&m.data.email).Validate(validateEmailField),
		huh.NewInput().
			Title("Mot de passe").
			EchoMode(huh.EchoModePassword).
			CharLimit(72).
			Value(&m.data.password).
			Validate(validatePasswordField),
		huh.NewSelect[string]().Title("Rôle").Options(roles...).Value(&m.data.role),
		huh.NewConfirm().
			Title("Compte confirmé").
			Affirmative("Oui").
			Negative("Non").
			Value(&m.data.confirmed),
	))
}

func (m *model) editForm() *huh.Form {
	roles := huh.NewOptions("client", "admin")
	return m.finish(huh.NewGroup(
		huh.NewInput().Title("Email").CharLimit(254).Value(&m.data.email).Validate(validateEmailField),
		huh.NewSelect[string]().Title("Rôle").Options(roles...).Value(&m.data.role),
		huh.NewInput().
			Title("Nouveau mot de passe").
			Description("Laisser vide pour conserver l'actuel.").
			EchoMode(huh.EchoModePassword).
			CharLimit(72).
			Value(&m.data.password).
			Validate(optionalPassword),
	))
}

func (m *model) deleteForm() *huh.Form {
	return m.finish(huh.NewGroup(
		huh.NewConfirm().
			Title("Supprimer " + m.selected.Email + " ?").
			Affirmative("Supprimer").
			Negative("Annuler").
			Value(&m.data.confirmed),
	))
}

func (m *model) finish(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).
		WithTheme(frame.Admin.FormTheme()).
		WithShowHelp(false).
		WithWidth(m.contentWidth())
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

func optionalPassword(value string) error {
	if value == "" {
		return nil
	}
	return validatePasswordField(value)
}
