package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/vmporuri/prompt-and-paint/internal/game"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func setupWSOriginCheck(cfg *Config) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		origin := r.Header.Get("origin")
		return origin == fmt.Sprintf("http://%s:%s", cfg.Server.Host, cfg.Server.Port)
	}
}

func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	defer conn.Close()
	client := &game.Client{Conn: conn}

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
