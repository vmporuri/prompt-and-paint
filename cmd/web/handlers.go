package main

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/google/uuid"
)

// Adds safe headers to HTTP responses.
func addSafeHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Strict-Transport-Security", "max-age=2592000; includeSubDomains")
	w.Header().Set("Cache-Control", "max-age=2592000")
}

// Registers the API endpoints for the server.
func registerRoutes(mux *http.ServeMux) {
	// Handles GET requests to the top level path.
	mux.HandleFunc("/{$}", func(w http.ResponseWriter, r *http.Request) {
		addSafeHeaders(w)
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

	// Handles WebSocket upgrade requests to the game endpoint.
	mux.HandleFunc("/game", func(w http.ResponseWriter, r *http.Request) {
		addSafeHeaders(w)
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		handleWS(w, r)
	})
}
