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
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		_, err := r.Cookie(jwtCookie)
		if err != nil {
			token, err := makeUserIDToken(uuid.NewString())
			if err != nil {
				log.Printf("Error printing %v", err)
			} else {
				http.SetCookie(w, &http.Cookie{
					Name:     jwtCookie,
					Value:    token,
					Path:     "/",
					MaxAge:   3600,
					Secure:   true,
					SameSite: http.SameSiteStrictMode,
				})
			}
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
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		handleWS(w, r)
	})
}
