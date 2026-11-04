package admintui

import "github.com/charmbracelet/glamour"

const helpMarkdown = `Cette application est séparée du client. Elle parle à la même API HTTP, avec un compte dont le rôle est admin.

## Comptes

Depuis la liste, vous pouvez créer un compte, changer son email ou son rôle, confirmer une inscription sans code, ou supprimer le compte.

Le mot de passe laissé vide lors d'une modification ne change pas l'empreinte. Un nouveau mot de passe ferme les sessions de ce compte.

Vous ne pouvez pas supprimer votre propre compte, ni retirer le dernier administrateur.

## SSH

make admin-sshd ouvre cet écran sur le port 2223. Le client reste sur le port 2222.
`

func renderMarkdown(source string, width int) string {
	if width < 20 {
		width = 20
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return source
	}
	rendered, err := renderer.Render(source)
	if err != nil || rendered == "" {
		return source
	}
	return rendered
}
