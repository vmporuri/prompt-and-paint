package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/olahol/melody"
)

type waitingRoomInfo struct {
	RoomID string
	IsHost bool
}

type websocketMessage struct {
	Headers map[string]any `json:"HEADERS"`
	Event   string         `json:"event"`
	Msg     string         `json:"msg"`
}

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

	mux.HandleFunc("/create-room", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(filepath.Join("templates", "waiting-room.html"))
		if err != nil {
			log.Println(err)
		}
		err = tmpl.Execute(w, waitingRoomInfo{
			RoomID: "123",
			IsHost: true,
		})
		if err != nil {
			log.Println(err)
		}
	})

	mux.HandleFunc("/join-room", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(filepath.Join("templates", "waiting-room.html"))
		if err != nil {
			log.Println(err)
		}
		err = tmpl.Execute(w, waitingRoomInfo{
			RoomID: "123",
			IsHost: false,
		})
		if err != nil {
			log.Println(err)
		}
	})

	mux.HandleFunc("/game", func(w http.ResponseWriter, r *http.Request) {
		err := m.HandleRequest(w, r)
		if err != nil {
			log.Println(err)
		}
	})
}

func registerWebsocketHandlers(m *melody.Melody) {
	m.HandleConnect(func(s *melody.Session) {
		s.Set("roomID", 123)
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		var wsMsg websocketMessage
		err := json.Unmarshal(msg, &wsMsg)
		if err != nil {
			log.Println(err)
		}
		s.Set("username", wsMsg.Msg)
	})
}
