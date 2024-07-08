package main

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/google/uuid"
)

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/{$}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		_, err := r.Cookie("userID")
		if err != nil {
			http.SetCookie(w, &http.Cookie{
				Name:     "userID",
				Value:    uuid.NewString(),
				Path:     "/",
				MaxAge:   3600,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			})
		}
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
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		handleWS(w, r)
	})
}
