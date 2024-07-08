package main

import (
	"errors"
	"log"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/vmporuri/prompt-and-paint/internal/game"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func setupWSOriginCheck(cfg *Config) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		origin, err := url.Parse(r.Header.Get("origin"))
		if err != nil {
			return false
		}
		return origin.Hostname() == cfg.Server.Host
	}
}

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

func writePump(client *game.Client) {
	for msg := range client.WriteChan {
		err := client.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println(err)
			return
		}
	}
}
