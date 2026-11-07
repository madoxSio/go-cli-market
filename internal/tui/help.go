package tui

import "github.com/charmbracelet/glamour"

const helpMarkdown = `Le client envoie chaque action au serveur HTTP. Le mot de passe n'est jamais stocké en clair : le serveur conserve une empreinte bcrypt.

## Inscription

1. Créez un compte avec un email et un mot de passe d'au moins 8 caractères.
2. Un code aléatoire s'affiche. Il remplace l'email de confirmation.
3. Confirmez le compte avec ce code. Sans confirmation, la connexion est refusée.
4. Le code expire au bout de 15 minutes. « Renvoyer le code » en génère un nouveau.

## Mot de passe oublié

La réinitialisation ne demande pas d'être connecté. Un code temporaire s'affiche, puis vous choisissez un nouveau mot de passe. Les sessions déjà ouvertes sont fermées.

## Catalogue

Une fois connecté, « Rechercher » filtre par nom, description, catégorie, prix HT ou prix TTC. Tous les champs sont facultatifs. Les prix se saisissent en euros, par exemple 49,99.

## Panier et paiement

Depuis une fiche produit, ajoutez une quantité. Le panier porte un identifiant BSK- et reste lié au compte. La livraison est offerte. Vous pouvez changer une quantité, retirer un produit et payer avec le numéro de carte, la date d'expiration et le CVC.

## Commandes

« Mes commandes » liste chaque commande avec sa date, son total et son statut : en attente, payée, en cours de livraison, livrée ou annulée. Le détail montre les produits et, en cas d'annulation, la raison. L'identifiant commence par CMD-.

## Session

La connexion renvoie un jeton. Le client le renvoie dans l'en-tête Authorization jusqu'à la déconnexion.

## SSH

La commande make sshd ouvre ce même écran via SSH. L'authentification du compte reste celle du formulaire, pas celle de la connexion SSH.
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
