# Go 1.27+ est requis (voir go.mod). Une toolchain posée dans .toolchain/ est utilisée si elle existe.
LOCAL_GO := $(CURDIR)/.toolchain/go/bin/go
ifneq ($(wildcard $(LOCAL_GO)),)
GO := $(LOCAL_GO)
export GOTOOLCHAIN := local
else
GO := go
endif

DATABASE_URL ?= postgres://market:market@localhost:5432/market?sslmode=disable
HTTP_ADDR ?= :8080

.PHONY: help db-up db-down db-reset db-wait db-logs db-psql server build test vet fmt tidy

.DEFAULT_GOAL := help

help:
	@echo "Base de données"
	@echo "  make db-up      Démarre PostgreSQL (Docker)"
	@echo "  make db-down    Arrête PostgreSQL"
	@echo "  make db-reset   Supprime le volume et recrée la base"
	@echo "  make db-wait    Attend que PostgreSQL accepte les connexions"
	@echo "  make db-logs    Suit les logs PostgreSQL"
	@echo "  make db-psql    Ouvre psql dans le conteneur"
	@echo ""
	@echo "Serveur HTTP"
	@echo "  make server     Lance l'API sur $(HTTP_ADDR)"
	@echo ""
	@echo "Outils"
	@echo "  make build      Compile le serveur dans bin/"
	@echo "  make test       Lance les tests"
	@echo "  make vet        Lance go vet"
	@echo "  make fmt        Formate le code"
	@echo "  make tidy       Range go.mod"

db-up:
	docker compose up -d db

db-down:
	docker compose down

db-reset:
	docker compose down -v
	docker compose up -d db

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

server:
	DATABASE_URL='$(DATABASE_URL)' HTTP_ADDR='$(HTTP_ADDR)' $(GO) run ./cmd/server

build:
	mkdir -p bin
	$(GO) build -o bin/server ./cmd/server

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

tidy:
	$(GO) mod tidy
