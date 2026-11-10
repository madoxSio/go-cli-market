package admintui

import "github.com/charmbracelet/glamour"

const helpMarkdown = `Cette application est séparée du client. Elle parle à la même API HTTP, avec un compte dont le rôle est admin.

## Comptes

Depuis la liste, vous pouvez créer un compte, changer son email ou son rôle, confirmer une inscription sans code, ou supprimer le compte.

Le mot de passe laissé vide lors d'une modification ne change pas l'empreinte. Un nouveau mot de passe ferme les sessions de ce compte.

Vous ne pouvez pas supprimer votre propre compte, ni retirer le dernier administrateur.

## Produits

« Nouveau produit » envoie le nom, la catégorie, le prix HT et le taux de TVA. Le serveur attribue l'identifiant PDT- et calcule le prix TTC.

## Commandes

« Commandes » liste les commandes de tous les clients. Une commande ouverte montre ses produits, son total et son statut.

« Nouvelle commande » choisit un client, un ou plusieurs produits et une quantité. La commande est créée en attente avec un identifiant CMD-.

« Changer le statut » ne propose que les statuts possibles : en attente vers payée ou annulée, payée vers en cours de livraison ou annulée, en cours de livraison vers livrée. Une annulation demande une raison, que le client voit dans sa commande.

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
