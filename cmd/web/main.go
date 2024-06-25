package main

import (
	"log"
	"net/http"

	"github.com/olahol/melody"
)

func main() {
	mux := http.NewServeMux()
	m := melody.New()

	registerRoutes(mux, m)
	registerWebsocketHandlers(m)
	log.Fatal(http.ListenAndServe(":3000", mux))
}
