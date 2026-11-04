# go-cli-market

Client d'achat en ligne de commande, API HTTP (`net/http`) et PostgreSQL (`database/sql`).

Go 1.27 ou plus récent est requis. Docker sert à lancer PostgreSQL.

## Lancer

Trois terminaux :

```bash
make db-up
make server
make client
```

Accès SSH au même client, une fois l'API démarrée :

```bash
make sshd
ssh -p 2222 localhost
```

`make help` liste les commandes base, serveur, client et SSH.

L'inscription affiche un code de confirmation : le sujet ne prévoit pas d'envoi d'email. Le mot de passe est stocké sous forme d'empreinte bcrypt. La connexion renvoie un jeton de session.
