// Commande admin lance l'interface terminal d'administration.
package main

import (
	"fmt"
	"os"

	"github.com/madoxSio/go-cli-market/internal/admintui"
	"github.com/madoxSio/go-cli-market/internal/config"
)

func main() {
	cfg := config.Load()
	if err := admintui.Run(cfg.APIURL); err != nil {
		fmt.Fprintf(os.Stderr, "admin : %v\n", err)
		os.Exit(1)
	}
}
