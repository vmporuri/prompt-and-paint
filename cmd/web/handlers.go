package main

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/olahol/melody"
)

func registerRoutes(mux *http.ServeMux, m *melody.Melody) {
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
		err := m.HandleRequest(w, r)
		if err != nil {
			log.Println(err)
		}
	})
}
