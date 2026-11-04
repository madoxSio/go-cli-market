package frame

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestScreenFitsTheTerminal(t *testing.T) {
	view := Client.Screen(80, 24, "Bienvenue", "Plateforme d'achat", "Session ouverte", true, "corps", Client.Keys("q", "quitter"))
	if lipgloss.Height(view) != 24 {
		t.Fatalf("hauteur %d, attendu 24", lipgloss.Height(view))
	}
	if lipgloss.Width(view) > 80 {
		t.Fatalf("largeur %d, attendu au plus 80", lipgloss.Width(view))
	}
	plain := stripANSI(view)
	if !strings.Contains(plain, "CLIENT") || !strings.Contains(plain, "Bienvenue") || !strings.Contains(plain, "quitter") {
		t.Fatalf("cadre incomplet:\n%s", plain)
	}
}

func TestGalleryStaysInsideItsBox(t *testing.T) {
	choices := []Choice{
		{Title: "Se connecter", Detail: "Email et mot de passe"},
		{Title: "Créer un compte", Detail: "Inscription, puis un code"},
		{Title: "Confirmer", Detail: "Saisir le code d'inscription"},
		{Title: "Renvoyer le code", Detail: "Si le précédent a expiré"},
		{Title: "Mot de passe oublié", Detail: "Réinitialiser sans connexion"},
		{Title: "Aide", Detail: "Parcours d'authentification"},
		{Title: "Quitter", Detail: "Fermer l'application"},
	}
	view := Client.Gallery(choices, 0, 72, 16)
	if lipgloss.Width(view) > 72 {
		t.Fatalf("largeur %d", lipgloss.Width(view))
	}
	if lipgloss.Height(view) > 16 {
		t.Fatalf("hauteur %d", lipgloss.Height(view))
	}
	plain := stripANSI(view)
	if !strings.Contains(plain, "Se connecter") || !strings.Contains(plain, "01") || !strings.Contains(plain, "╭") {
		t.Fatalf("carte absente:\n%s", plain)
	}
}

func TestAdminScreenFitsTheTerminal(t *testing.T) {
	view := Admin.Screen(100, 30, "Connexion", "Réservé aux administrateurs", "", false, "formulaire", Admin.Keys("entrée", "valider"))
	if lipgloss.Height(view) != 30 {
		t.Fatalf("hauteur %d, attendu 30", lipgloss.Height(view))
	}
	if lipgloss.Width(view) > 100 {
		t.Fatalf("largeur %d, attendu au plus 100", lipgloss.Width(view))
	}
}

func stripANSI(value string) string {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] != 0x1b {
			b.WriteByte(value[i])
			continue
		}
		for i < len(value) && value[i] != 'm' {
			i++
		}
	}
	return b.String()
}
