package main

import (
	"encoding/json"
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

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		var gameMsg game.GameMessage
		err = json.Unmarshal(p, &gameMsg)
		if err != nil {
			log.Println(err)
		}

		game.DispatchGameEvent(client, &gameMsg)
	}
}
