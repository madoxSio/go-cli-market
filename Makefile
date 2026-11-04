# Go 1.27+ est requis (voir go.mod). Une toolchain posée dans .toolchain/ est utilisée si elle existe.
LOCAL_GO := $(CURDIR)/.toolchain/go/bin/go
ifneq ($(wildcard $(LOCAL_GO)),)
GO := $(LOCAL_GO)
export GOTOOLCHAIN := local
else
GO := go
endif

DATABASE_URL ?= postgres://market:market@localhost:5432/market?sslmode=disable
TEST_DATABASE_URL ?= $(DATABASE_URL)
HTTP_ADDR ?= :8080
API_URL ?= http://localhost:8080
SSH_ADDR ?= :2222
SSH_HOST_KEY ?= .ssh/server_ed25519
ADMIN_SSH_ADDR ?= :2223
ADMIN_SSH_HOST_KEY ?= .ssh/admin_ed25519

# Serveur HTTP lancé en arrière-plan par make up : journal et numéro de processus.
RUN_DIR := .run
SERVER_LOG := $(RUN_DIR)/server.log
SERVER_PID := $(RUN_DIR)/server.pid

.PHONY: help up down restart status logs \
	db-up db-down db-reset db-wait db-logs db-psql \
	server server-bg server-stop require-server \
	client admin sshd admin-sshd ssh-client ssh-admin \
	build test test-db vet fmt check tidy

.DEFAULT_GOAL := help

help:
	@echo "Démarrage rapide"
	@echo "  make up          Base + serveur HTTP en arrière-plan (tout ce qu'il faut)"
	@echo "  make client      App client dans ce terminal"
	@echo "  make admin       App admin dans ce terminal (compte : $${ADMIN_EMAIL:-admin@market.local} / $${ADMIN_PASSWORD:-adminadmin})"
	@echo "  make status      Ce qui tourne"
	@echo "  make logs        Suit le journal du serveur HTTP"
	@echo "  make restart     Relance le serveur HTTP (après une modification du code)"
	@echo "  make down        Arrête le serveur HTTP et la base"
	@echo ""
	@echo "SSH (après make up)"
	@echo "  make sshd        Serveur SSH du client sur $(SSH_ADDR)   puis : make ssh-client"
	@echo "  make admin-sshd  Serveur SSH de l'admin sur $(ADMIN_SSH_ADDR) puis : make ssh-admin"
	@echo ""
	@echo "Base de données"
	@echo "  make db-up       Démarre PostgreSQL (Docker)"
	@echo "  make db-down     Arrête PostgreSQL"
	@echo "  make db-reset    Supprime le volume et recrée la base"
	@echo "  make db-wait     Attend que PostgreSQL accepte les connexions"
	@echo "  make db-logs     Suit les logs PostgreSQL"
	@echo "  make db-psql     Ouvre psql dans le conteneur"
	@echo ""
	@echo "Serveur HTTP"
	@echo "  make server      Lance l'API au premier plan sur $(HTTP_ADDR) (Ctrl+C pour arrêter)"
	@echo ""
	@echo "Outils"
	@echo "  make check       gofmt + go vet + go test (à lancer avant chaque commit)"
	@echo "  make test        Lance les tests (sans base : les tests store sont ignorés)"
	@echo "  make test-db     Lance les tests, y compris ceux contre PostgreSQL"
	@echo "  make build       Compile server, client, sshd, admin et admin-sshd dans bin/"
	@echo "  make vet         Lance go vet"
	@echo "  make fmt         Formate le code"
	@echo "  make tidy        Range go.mod"

# --- Démarrage rapide -------------------------------------------------------

up: db-up db-wait server-bg
	@echo ""
	@echo "Prêt. Dans un autre terminal : make client  ou  make admin"

down: server-stop db-down

restart: server-stop server-bg

status:
	@docker compose ps
	@echo ""
	@if curl -sf $(API_URL)/health >/dev/null 2>&1; then \
		echo "Serveur HTTP : en marche sur $(API_URL)"; \
	else \
		echo "Serveur HTTP : arrêté (make up)"; \
	fi

logs:
	@test -f $(SERVER_LOG) || { echo "Pas de journal : le serveur n'a pas été lancé avec make up"; exit 1; }
	tail -f $(SERVER_LOG)

# --- Base de données --------------------------------------------------------

db-up:
	docker compose up -d db

db-down:
	docker compose down

db-reset:
	docker compose down -v
	docker compose up -d db

#voir la disponibilité de la base de données.
db-wait:
	@for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do \
		docker compose exec -T db pg_isready -U market -d market && exit 0; \
		sleep 1; \
	done; \
	echo "PostgreSQL ne répond pas"; exit 1

db-logs:
	docker compose logs -f db

db-psql:
	docker compose exec db psql -U market -d market

# --- Serveur HTTP -----------------------------------------------------------

server:
	DATABASE_URL='$(DATABASE_URL)' HTTP_ADDR='$(HTTP_ADDR)' $(GO) run ./cmd/server

# Compile puis lance le serveur en arrière-plan, et attend qu'il réponde sur /health.
server-bg:
	@if curl -sf $(API_URL)/health >/dev/null 2>&1; then \
		echo "Serveur HTTP déjà en marche sur $(API_URL)"; \
	else \
		mkdir -p bin $(RUN_DIR); \
		$(GO) build -o bin/server ./cmd/server || exit 1; \
		DATABASE_URL='$(DATABASE_URL)' HTTP_ADDR='$(HTTP_ADDR)' nohup bin/server > $(SERVER_LOG) 2>&1 & \
		echo $$! > $(SERVER_PID); \
		for i in 1 2 3 4 5 6 7 8 9 10; do \
			sleep 1; \
			curl -sf $(API_URL)/health >/dev/null 2>&1 && { echo "Serveur HTTP démarré sur $(API_URL) (journal : make logs)"; exit 0; }; \
		done; \
		echo "Le serveur ne répond pas. Dernières lignes du journal :"; tail -n 20 $(SERVER_LOG); exit 1; \
	fi

server-stop:
	@if [ -f $(SERVER_PID) ] && kill $$(cat $(SERVER_PID)) 2>/dev/null; then \
		echo "Serveur HTTP arrêté"; \
	else \
		echo "Aucun serveur HTTP lancé par make up"; \
	fi
	@rm -f $(SERVER_PID)

require-server:
	@curl -sf $(API_URL)/health >/dev/null 2>&1 || { \
		echo "Le serveur HTTP ne répond pas sur $(API_URL)."; \
		echo "Lancez d'abord : make up"; \
		exit 1; \
	}

# --- Applications -----------------------------------------------------------

client: require-server
	API_URL='$(API_URL)' $(GO) run ./cmd/client

admin: require-server
	API_URL='$(API_URL)' $(GO) run ./cmd/admin

sshd: require-server
	API_URL='$(API_URL)' SSH_ADDR='$(SSH_ADDR)' SSH_HOST_KEY='$(SSH_HOST_KEY)' $(GO) run ./cmd/sshd

admin-sshd: require-server
	API_URL='$(API_URL)' ADMIN_SSH_ADDR='$(ADMIN_SSH_ADDR)' ADMIN_SSH_HOST_KEY='$(ADMIN_SSH_HOST_KEY)' $(GO) run ./cmd/admin-sshd

ssh-client:
	ssh -p $(lastword $(subst :, ,$(SSH_ADDR))) localhost

ssh-admin:
	ssh -p $(lastword $(subst :, ,$(ADMIN_SSH_ADDR))) localhost

# --- Outils -----------------------------------------------------------------

build:
	mkdir -p bin
	$(GO) build -o bin/server ./cmd/server
	$(GO) build -o bin/client ./cmd/client
	$(GO) build -o bin/sshd ./cmd/sshd
	$(GO) build -o bin/admin ./cmd/admin
	$(GO) build -o bin/admin-sshd ./cmd/admin-sshd

test:
	$(GO) test ./...

test-db:
	TEST_DATABASE_URL='$(TEST_DATABASE_URL)' $(GO) test -count=1 ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

check:
	@test -z "$$(gofmt -l .)" || { echo "Fichiers mal formatés (make fmt) :"; gofmt -l .; exit 1; }
	$(GO) vet ./...
	$(GO) test ./...

tidy:
	$(GO) mod tidy
