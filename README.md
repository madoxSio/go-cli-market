# go-cli-market

API HTTP (`net/http`), client et administration en ligne de commande, PostgreSQL (`database/sql`).

Go 1.27 ou plus récent est requis. Docker sert à lancer PostgreSQL.

## Lancer

```bash
make db-up
make server
```

Puis, dans d'autres terminaux :

```bash
make client
make admin
```

SSH, une fois l'API démarrée :

```bash
make sshd
ssh -p 2222 localhost

make admin-sshd
ssh -p 2223 localhost
```

Le compte administrateur est créé au premier démarrage s'il n'en existe pas : `admin@market.local` / `adminadmin`. On le change avec `ADMIN_EMAIL` et `ADMIN_PASSWORD`.

`make help` liste les commandes.

L'inscription client affiche un code de confirmation : le sujet ne prévoit pas d'envoi d'email. Le mot de passe est stocké sous forme d'empreinte bcrypt. La connexion renvoie un jeton de session.
