package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	registerRoutes(mux)
	log.Fatal(http.ListenAndServe(":3000", mux))
}
