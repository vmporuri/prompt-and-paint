package main

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(filepath.Join("templates", "index.html"))
		if err != nil {
			log.Fatalf("Error parsing template: %v", err)
		}
		err = tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, "Unable to render template", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/game", func(w http.ResponseWriter, r *http.Request) {
		handleWS(w, r)
	})
}
