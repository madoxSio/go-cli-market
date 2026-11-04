// Package config lit la configuration dans les variables d'environnement.
package config

import "os"

// Config rassemble les adresses du serveur, du client et de PostgreSQL.
type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	APIURL          string
	SSHAddr         string
	SSHHostKey      string
	AdminSSHAddr    string
	AdminSSHHostKey string
	AdminEmail      string
	AdminPassword   string
}

// Load lit la configuration. Chaque valeur a un défaut adapté au développement local.
func Load() Config {
	return Config{
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:     getenv("DATABASE_URL", "postgres://market:market@localhost:5432/market?sslmode=disable"),
		APIURL:          getenv("API_URL", "http://localhost:8080"),
		SSHAddr:         getenv("SSH_ADDR", ":2222"),
		SSHHostKey:      getenv("SSH_HOST_KEY", ".ssh/server_ed25519"),
		AdminSSHAddr:    getenv("ADMIN_SSH_ADDR", ":2223"),
		AdminSSHHostKey: getenv("ADMIN_SSH_HOST_KEY", ".ssh/admin_ed25519"),
		AdminEmail:      getenv("ADMIN_EMAIL", "admin@market.local"),
		AdminPassword:   getenv("ADMIN_PASSWORD", "adminadmin"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
