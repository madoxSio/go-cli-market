package store

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"

	"github.com/madoxSio/go-cli-market/internal/market"
)

// Market regroupe le SQL du catalogue, des paniers et des commandes.
// (Repository, à côté, gère les comptes et les sessions pour internal/auth.)
type Market struct {
	db *sql.DB
}

// NewMarket branche le catalogue, les paniers et les commandes sur la base ouverte par Open.
func NewMarket(db *sql.DB) *Market {
	return &Market{db: db}
}

// newRef tire un identifiant métier. C'est une variable pour que les tests puissent
// la remplacer et provoquer volontairement une collision.
var newRef = market.NewBusinessID

// maxRefAttempts est le nombre d'essais quand un identifiant tiré existe déjà.
const maxRefAttempts = 5

// isUniqueViolation indique si err vient d'une contrainte UNIQUE précise (code PostgreSQL 23505).
func isUniqueViolation(err error, constraint string) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505" && pqErr.Constraint == constraint
}

// isForeignKeyViolation indique si err vient d'une clé étrangère vers une ligne absente (code 23503).
func isForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23503"
}
