// Package config lit la configuration dans les variables d'environnement.
package config

import "os"

// Config rassemble les adresses du serveur, du client et de PostgreSQL.
type Config struct {
	HTTPAddr    string
	DatabaseURL string
	APIURL      string
	SSHAddr     string
	SSHHostKey  string
}

// Load lit la configuration. Chaque valeur a un défaut adapté au développement local.
func Load() Config {
	return Config{
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://market:market@localhost:5432/market?sslmode=disable"),
		APIURL:      getenv("API_URL", "http://localhost:8080"),
		SSHAddr:     getenv("SSH_ADDR", ":2222"),
		SSHHostKey:  getenv("SSH_HOST_KEY", ".ssh/server_ed25519"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
