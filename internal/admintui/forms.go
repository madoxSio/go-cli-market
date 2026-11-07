package admintui

import (
	"errors"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/madoxSio/go-cli-market/internal/frame"
	"github.com/madoxSio/go-cli-market/internal/market"
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

func (m *model) productForm() *huh.Form {
	return m.finish(huh.NewGroup(
		huh.NewInput().
			Title("Nom").
			CharLimit(200).
			Value(&m.data.name).
			Validate(requiredField("nom requis")),
		huh.NewInput().
			Title("Description").
			Description("Facultative.").
			CharLimit(2000).
			Value(&m.data.description),
		huh.NewInput().
			Title("Catégorie").
			CharLimit(100).
			Value(&m.data.category).
			Validate(requiredField("catégorie requise")),
		huh.NewInput().
			Title("Prix HT").
			Description("En euros, exemple 49,99. La TVA est ajoutée ensuite.").
			Value(&m.data.price).
			Validate(func(value string) error {
				cents, err := market.ParseEuros(value)
				if err != nil {
					return err
				}
				if cents <= 0 {
					return errors.New("le prix doit être supérieur à 0")
				}
				return nil
			}),
		huh.NewSelect[int64]().
			Title("TVA").
			Options(
				huh.NewOption("20 %", int64(market.VATStandard)),
				huh.NewOption("10 %", int64(market.VATIntermediate)),
				huh.NewOption("5,5 %", int64(market.VATReduced)),
				huh.NewOption("2,1 %", int64(market.VATSuper)),
			).
			Value(&m.data.vat),
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

func requiredField(message string) func(string) error {
	return func(value string) error {
		if strings.TrimSpace(value) == "" {
			return errors.New(message)
		}
		return nil
	}
}

func optionalPassword(value string) error {
	if value == "" {
		return nil
	}
	return validatePasswordField(value)
}
