package game

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"path/filepath"
)

type gamePageData struct {
	Question string
}

type votingPageData struct {
	URLs []string
}

type imagePreviewData struct {
	URL string
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
	buf.Write([]byte(fmt.Sprintf("%s:", newPlayerList)))
	return generateTemplate(buf, filepath.Join("templates", "player-list.html"), room)
}

func generateGamePage(gpd *gamePageData) []byte {
	buf := &bytes.Buffer{}
	buf.Write([]byte(fmt.Sprintf("%s:", enterGame)))
	return generateTemplate(buf, filepath.Join("templates", "game-page.html"), gpd)
}

func generateVotingPage(vpd *votingPageData) []byte {
	buf := &bytes.Buffer{}
	buf.Write([]byte(fmt.Sprintf("%s:", votePage)))
	return generateTemplate(buf, filepath.Join("templates", "voting-page.html"), vpd)
}

func generatePicturePreview(ipd *imagePreviewData) []byte {
	return generateTemplate(nil, filepath.Join("templates", "image-preview.html"), ipd)
}
