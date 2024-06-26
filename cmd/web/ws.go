package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log"
	"path/filepath"

	"github.com/olahol/melody"
)

type websocketMessage struct {
	Headers map[string]any `json:"HEADERS"`
	Event   websocketEvent `json:"event"`
	Msg     string         `json:"msg"`
}

type websocketEvent string

const (
	setUsername websocketEvent = "set-username"
	ready       websocketEvent = "ready"
)

func registerWebsocketHandlers(m *melody.Melody) {
	m.HandleConnect(handleWSConnection)

	m.HandleMessage(handleGameMessage)
}

func handleWSConnection(s *melody.Session) {
	log.Println("New connection!")
}

func handleGameMessage(s *melody.Session, msg []byte) {
	var wsMsg websocketMessage
	err := json.Unmarshal(msg, &wsMsg)
	if err != nil {
		log.Println(err)
	}

	switch wsMsg.Event {
	case setUsername:
		s.Set("username", wsMsg.Msg)
		createRoom(s)
		log.Println("User joined game.")

		waitingRoom := &bytes.Buffer{}
		tmpl, err := template.ParseFiles(filepath.Join("templates", "waiting-room.html"))
		if err != nil {
			log.Println(err)
		}
		err = tmpl.Execute(waitingRoom, nil)
		if err != nil {
			log.Println(err)
		}
		err = s.Write(waitingRoom.Bytes())
		if err != nil {
			log.Println(err)
		}
	}
}
