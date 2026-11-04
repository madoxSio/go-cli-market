// Package store ouvre PostgreSQL et applique le schéma avec database/sql.
package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

//go:embed schema.sql
var schemaFS embed.FS

// Open ouvre un pool database/sql et vérifie que PostgreSQL répond.
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping PostgreSQL : %w", err)
	}
	return db, nil
}

// Migrate crée les tables manquantes. Le script est idempotent.
func Migrate(db *sql.DB) error {
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return err
	}
	if _, err := db.Exec(string(schema)); err != nil {
		return fmt.Errorf("schéma : %w", err)
	}
	return nil
}
