package game

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
)

func generateTemplate(buf *bytes.Buffer, path string, data any) ([]byte, error) {
	if buf == nil {
		buf = &bytes.Buffer{}
	}
	tmpl, err := template.ParseFiles(path)
	if err != nil {
		return nil, err
	}
	err = tmpl.Execute(buf, data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func generateUsername() ([]byte, error) {
	return generateTemplate(nil, filepath.Join("templates", "username.html"), nil)
}

func generateWaitingPage(wpd *waitingPageData) ([]byte, error) {
	return generateTemplate(nil, filepath.Join("templates", "waiting-page.html"), wpd)
}

func generatePlayerList(pld *playerListData) ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.Write([]byte(fmt.Sprintf("%s:", newPlayerList)))
	return generateTemplate(buf, filepath.Join("templates", "player-list.html"), pld)
}

func generateGamePage(gpd *gamePageData) ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.Write([]byte(fmt.Sprintf("%s:", enterGame)))
	return generateTemplate(buf, filepath.Join("templates", "game-page.html"), gpd)
}

func generateVotingPage(vpd *votingPageData) ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.Write([]byte(fmt.Sprintf("%s:", votePage)))
	return generateTemplate(buf, filepath.Join("templates", "voting-page.html"), vpd)
}

func generatePicturePreview(ipd *imagePreviewData) ([]byte, error) {
	return generateTemplate(nil, filepath.Join("templates", "image-preview.html"), ipd)
}

func generateLeaderboardPage(lpd *leaderboardPageData) ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.Write([]byte(fmt.Sprintf("%s:", leaderboard)))
	return generateTemplate(buf, filepath.Join("templates", "leaderboard.html"), lpd)
}
