// Commande client lance l'interface terminal de l'acheteur.
package main

import (
	"fmt"
	"os"

	"github.com/madoxSio/go-cli-market/internal/config"
	"github.com/madoxSio/go-cli-market/internal/tui"
)

func main() {
	cfg := config.Load()
	if err := tui.Run(cfg.APIURL); err != nil {
		fmt.Fprintf(os.Stderr, "client : %v\n", err)
		os.Exit(1)
	}
}
