package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/vmporuri/prompt-and-paint/internal/game"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Adds the origin check for the WebSocket upgrade request.
// If the origin does not match, does not upgrade the connection.
func setupWSOriginCheck(cfg *Config) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		origin := r.Header.Get("origin")
		for _, allowed := range cfg.Security.AllowedOrigins {
			if origin == allowed {
				return true
			}
		}
		return false
	}
}

// Gets the userID from the cookie in the HTTP request.
// Errors if there is no cookie or the cookie is not formatted properly.
func getCookie(r *http.Request) (string, error) {
	tokenString, err := r.Cookie(jwtCookie)
	if err != nil {
		return "", errors.New("No userID cookie present")
	}
	userID, err := parseUserIDToken(tokenString.Value)
	if err != nil {
		return "", err
	}
	return userID, nil
}

// Sets up the WebSocket connection and begins reading from it.
// Parses incoming messages as game events and processes via the game API.
// Closes the read and write pump upon disconnection.
func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	userID, err := getCookie(r)
	if err != nil {
		log.Println(err)
		userID = uuid.NewString()
	}
	client := game.NewClient(conn, userID)
	go writePump(client)

	for {
		var gameMsg game.GameMessage
		err := conn.ReadJSON(&gameMsg)
		if err != nil {
			log.Println(err)
			game.DispatchGameEvent(client, &game.GameMessage{Event: game.CloseWS})
			return
		}

		game.DispatchGameEvent(client, &gameMsg)
	}
}

// Continually checks for new messages to write to the WebSocket connection and sends
// them as they come in.
func writePump(client *game.Client) {
	for msg := range client.WriteChan {
		err := client.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println(err)
			return
		}
	}
}
