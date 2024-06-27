package game

import (
	"bytes"
	"html/template"
	"log"
	"path/filepath"
)

func generateTemplate(buf *bytes.Buffer, path string, data any) []byte {
	if buf == nil {
		buf = &bytes.Buffer{}
	}
	tmpl, err := template.ParseFiles(path)
	if err != nil {
		log.Println(err)
	}
	err = tmpl.Execute(buf, data)
	if err != nil {
		log.Println(err)
	}
	return buf.Bytes()
}

func generateUsername() []byte {
	return generateTemplate(nil, filepath.Join("templates", "username.html"), nil)
}

func generateWaitingRoom(client *Client) []byte {
	return generateTemplate(nil, filepath.Join("templates", "waiting-room.html"), client)
}

func generatePlayerList(room *Room) []byte {
	buf := &bytes.Buffer{}
	buf.Write([]byte("new-player-list:"))
	return generateTemplate(buf, filepath.Join("templates", "player-list.html"), *room)
}
