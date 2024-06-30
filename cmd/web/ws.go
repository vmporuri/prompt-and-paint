package main

import (
	"log"
	"net/http"
	"net/url"

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

func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	defer conn.Close()
	client := game.NewClient(conn)
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
