package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/vmporuri/prompt-and-paint/internal/game"
)

// Boots up the server.
func main() {
	mux := http.NewServeMux()
	registerRoutes(mux)

	readConfig()
	setupWSOriginCheck(&cfg)
	game.SetupDBConnection(createDBConnection(&cfg))

	log.Printf("Server listening on :%s", cfg.Server.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", cfg.Server.Port), mux))
}
