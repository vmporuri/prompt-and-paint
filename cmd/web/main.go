package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/vmporuri/prompt-and-paint/internal/game"
)

func main() {
	mux := http.NewServeMux()
	registerRoutes(mux)

	readConfig()
	setupWSOriginCheck(&cfg)
	game.SetupDBConnection(createDBConnection(&cfg))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", cfg.Server.Port), mux))
}
