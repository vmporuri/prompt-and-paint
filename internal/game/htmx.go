package game

import (
	"bytes"
	"html/template"
	"log"
	"path/filepath"
)

type gamePageData struct {
	Question string
}

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

func generateWaitingPage(client *Client) []byte {
	return generateTemplate(nil, filepath.Join("templates", "waiting-page.html"), client)
}

func generatePlayerList(room *Room) []byte {
	buf := &bytes.Buffer{}
	buf.Write([]byte("new-player-list:"))
	return generateTemplate(buf, filepath.Join("templates", "player-list.html"), room)
}

func generateGamePage(gpd *gamePageData) []byte {
	buf := &bytes.Buffer{}
	buf.Write([]byte("game-room:"))
	return generateTemplate(buf, filepath.Join("templates", "game-page.html"), gpd)
}
