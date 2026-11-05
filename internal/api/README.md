# Contrat de l'API HTTP

Ce document décrit toutes les routes du serveur (`cmd/server`) et le format de leurs données. Il fait référence pour le serveur (`internal/httpapi`) et pour les interfaces (`internal/apiclient`, `internal/tui`, `internal/admintui`). **Toute modification d'une route ou d'un format JSON se fait ici d'abord, et se signale à l'équipe.**

Les types Go correspondants sont dans `internal/market` (produit, panier, commande, carte) et dans `internal/api/api.go` (corps des requêtes).

## Conventions

| Sujet | Règle |
| --- | --- |
| Adresse | `http://localhost:8080` en développement (`API_URL`) |
| Format | JSON en entrée et en sortie (`Content-Type: application/json`), 1 Mo maximum par requête |
| Noms des champs | `snake_case` (`product_ref`, `price_ttc`) |
| **Montants** | **Entiers en centimes d'euro** : `4999` = 49,99 €. Jamais de nombre à virgule. Pour afficher : `fmt.Sprintf("%d,%02d €", c/100, c%100)` ou `market.Cents(c).String()` |
| **Taux de TVA** | **Points de base** : `2000` = 20 %, `550` = 5,5 % |
| Dates | Texte RFC 3339, en UTC (suffixe `Z`), avec fractions de seconde : `"2026-11-05T10:38:27.614055Z"`. À convertir en heure locale pour l'affichage (`t.Local()` en Go) |
| Identifiants métier | `PDT-XXXXXX` (produit), `BSK-XXXXXX` (panier), `CMD-XXXXXX` (commande) : 6 caractères `A-Z0-9` |
| Authentification | En-tête `Authorization: Bearer <token>`, le jeton renvoyé par `POST /auth/login` |
| Erreurs | Corps `{"error": "message lisible en français"}`, avec le code HTTP ci-dessous |

### Codes de réponse

| Code | Sens |
| --- | --- |
| 200 | OK |
| 201 | Créé (inscription, produit, commande) |
| 400 | Données invalides (JSON mal formé, champ manquant, quantité, carte refusée…) |
| 401 | Pas connecté, ou jeton invalide / expiré |
| 403 | Connecté, mais pas le droit (route admin avec un compte client, compte non confirmé) |
| 404 | Introuvable (produit, commande…) |
| 405 | Mauvaise méthode HTTP pour cette route |
| 409 | Conflit avec l'état actuel (email déjà pris, transition de statut interdite, panier vide…) |
| 500 | Erreur interne du serveur (le détail est dans les logs du serveur, jamais dans la réponse) |

---

## Santé

| Méthode | Route | Accès | Réponse |
| --- | --- | --- | --- |
| GET | `/health` | public | `{"status": "ok"}` |

---

## Authentification (fait, Mathurin)

| Méthode | Route | Accès | Corps de la requête | Réponse |
| --- | --- | --- | --- | --- |
| POST | `/auth/register` | public | `{"email", "password"}` | **201** `{"message", "confirmation_code"}` |
| POST | `/auth/confirm` | public | `{"email", "code"}` | `{"message"}` |
| POST | `/auth/confirmation-code` | public | `{"email"}` | `{"message", "confirmation_code"}` (code vide si aucun compte en attente) |
| POST | `/auth/login` | public | `{"email", "password"}` | `{"token", "user": User}` |
| POST | `/auth/forgot-password` | public | `{"email"}` | `{"message", "reset_code"}` (code vide si le compte n'existe pas) |
| POST | `/auth/reset-password` | public | `{"email", "code", "new_password"}` | `{"message"}` |
| POST | `/auth/logout` | connecté | — | `{"message"}` |
| GET | `/auth/me` | connecté | — | `User` |

`User` = `{"id": 6, "email": "ada@example.com", "role": "client" | "admin", "confirmed": true}`

Mot de passe : 8 à 72 caractères. Codes : 8 caractères, valables 15 minutes. Session : 24 heures. Un compte non confirmé ne peut pas se connecter (403).

## Administration des comptes (fait, Mathurin)

| Méthode | Route | Corps | Réponse |
| --- | --- | --- | --- |
| GET | `/admin/users` | — | `[User, …]` |
| POST | `/admin/users` | `{"email", "password", "role", "confirmed"}` | **201** `User` |
| PUT | `/admin/users/{id}` | `{"email", "password", "role"}` (mot de passe vide = inchangé) | `User` |
| DELETE | `/admin/users/{id}` | — | `{"message"}` |
| POST | `/admin/users/{id}/confirm` | — | `User` |

Toutes les routes `/admin/...` exigent un compte `admin` (sinon 401 ou 403).

---

## Catalogue (madox)

### `GET /products` (public) : recherche

Paramètres d'URL, **tous facultatifs** et combinables :

| Paramètre | Effet | Exemple |
| --- | --- | --- |
| `q` | Texte cherché dans le **nom** et la **description**, sans tenir compte des majuscules | `q=clavier` |
| `category` | Catégorie exacte (sans tenir compte des majuscules) | `category=informatique` |
| `min_price`, `max_price` | Fourchette de prix **HT**, en centimes, bornes incluses | `max_price=5000` |
| `min_ttc`, `max_ttc` | Fourchette de prix **TTC**, en centimes, bornes incluses | `min_ttc=1000&max_ttc=3000` |

Réponse : `[Product, …]` (liste vide `[]` si rien ne correspond), triée par nom. Un paramètre de prix qui n'est pas un entier positif → 400.

```
GET /products?q=clavier&max_ttc=10000
```

### `GET /products/{ref}` (public)

Réponse : `Product`, ou 404.

### `POST /admin/products` (admin) : ajouter un produit

Corps (`market.ProductInput`) :
```json
{"name": "Clavier mécanique", "description": "Switchs rouges", "category": "informatique",
 "price_ht": 4999, "vat_rate": 2000}
```
`vat_rate` est facultatif (20 % par défaut). Règles : nom obligatoire (200 caractères max), catégorie obligatoire (100 max, enregistrée en minuscules), description 2000 max, `price_ht` > 0, `vat_rate` de 0 à 10000.

Réponse : **201** `Product` (avec sa `ref` générée).

### `Product`

```json
{
  "ref": "PDT-7D2K8N",
  "name": "Clavier mécanique",
  "description": "Switchs rouges",
  "category": "informatique",
  "price_ht": 4999,
  "vat_rate": 2000,
  "price_ttc": 5999,
  "created_at": "2026-11-05T10:38:27.614055Z"
}
```
`price_ttc` est calculé par le serveur (arrondi au centime le plus proche).

---

## Panier (madox) : client connecté

Chaque client a **un panier en cours**, créé automatiquement au premier ajout. Après le paiement, il est figé et le prochain ajout crée un nouveau panier (nouvelle `ref`).

| Méthode | Route | Corps | Réponse |
| --- | --- | --- | --- |
| GET | `/cart` | — | `Cart` (vide si aucun panier en cours) |
| POST | `/cart/items` | `CartItemsRequest` : **ajoute** les quantités aux produits (un ou plusieurs) | `Cart` |
| PATCH | `/cart/items` | `CartItemsRequest` : **remplace** les quantités (un ou plusieurs) ; `quantity: 0` retire le produit | `Cart` |
| DELETE | `/cart/items/{productRef}` | — : retire un produit | `Cart` |
| POST | `/cart/checkout` | `CheckoutRequest` : paie le panier | **201** `Order` |

`CartItemsRequest` :
```json
{"items": [{"product_ref": "PDT-7D2K8N", "quantity": 2}, {"product_ref": "PDT-A1B2C3", "quantity": 1}]}
```
Quantité par produit : 1 à 99 (0 accepté seulement dans `PATCH`). Produit inconnu → 404.

`CheckoutRequest` :
```json
{"card": {"number": "4242 4242 4242 4242", "expiry": "12/28", "cvc": "123"}}
```
La carte est vérifiée (algorithme de Luhn, date non expirée, CVC de 3 chiffres) → 400 sinon. Panier vide → 409. Seuls les 4 derniers chiffres sont conservés. La commande créée a le statut `paid`.

### `Cart`

```json
{
  "ref": "BSK-Q4W8ZP",
  "items": [
    {"product_ref": "PDT-7D2K8N", "name": "Clavier mécanique", "unit_price_ht": 4999,
     "vat_rate": 2000, "unit_price_ttc": 5999, "quantity": 2, "line_total_ttc": 11998}
  ],
  "shipping": 0,
  "total_ttc": 11998
}
```
Le prix TTC est arrondi **par unité**, puis multiplié par la quantité. La livraison (`shipping`) est toujours gratuite. Panier vide : `{"ref": "", "items": [], "shipping": 0, "total_ttc": 0}`.

---

## Commandes (madox)

### Client connecté

| Méthode | Route | Réponse |
| --- | --- | --- |
| GET | `/orders` | `[Order, …]` : ses commandes, la plus récente d'abord |
| GET | `/orders/{ref}` | `Order` ; 404 si elle n'existe pas **ou appartient à un autre client** |

### Admin

| Méthode | Route | Corps | Réponse |
| --- | --- | --- | --- |
| GET | `/admin/orders` | — (paramètre facultatif `?status=paid` ; statut inconnu → 400) | `[Order, …]` de tous les clients, la plus récente d'abord |
| POST | `/admin/orders` | `CreateOrderRequest` : commande pour un client | **201** `Order` au statut `pending` |
| PATCH | `/admin/orders/{ref}/status` | `UpdateStatusRequest` | `Order` |

`CreateOrderRequest` :
```json
{"user_id": 6, "items": [{"product_ref": "PDT-7D2K8N", "quantity": 1}]}
```
`user_id` obligatoire (l'`id` renvoyé par `GET /admin/users`) ; client inconnu → 404. Un même produit cité deux fois voit ses quantités additionnées. Prix : ceux du catalogue au moment de la création.

`UpdateStatusRequest` :
```json
{"status": "cancelled", "reason": "Rupture de stock"}
```

### Statuts et transitions

| Statut | Libellé | Peut devenir |
| --- | --- | --- |
| `pending` | en attente | `paid`, `cancelled` |
| `paid` | payée | `shipping`, `cancelled` |
| `shipping` | en cours de livraison | `delivered` |
| `delivered` | livrée | — (final) |
| `cancelled` | annulée | — (final) |

Statut inconnu → 400. Transition interdite → 409. `reason` obligatoire pour `cancelled` → 400 sinon.

### `Order`

```json
{
  "ref": "CMD-7D2K8N",
  "user_id": 6,
  "user_email": "ada@example.com",
  "status": "cancelled",
  "cancel_reason": "Rupture de stock",
  "items": [ /* mêmes lignes que le panier, prix figés au moment de la commande */ ],
  "total_ttc": 11998,
  "card_last4": "4242",
  "created_at": "2026-11-05T10:46:53.743003Z",
  "updated_at": "2026-11-05T11:02:10.120544Z"
}
```
`user_email` est l'email du client. `cancel_reason` (si annulée) et `card_last4` (si payée par carte) sont absents quand ils sont vides. Les prix des lignes sont **copiés** au moment de la commande : un changement de prix du produit ne modifie pas l'historique.
