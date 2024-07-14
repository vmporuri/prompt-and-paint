package game

import (
	"bytes"
	"html/template"
	"path/filepath"
)

func generateTemplate(path string, data any) ([]byte, error) {
	buf := &bytes.Buffer{}
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
	return generateTemplate(filepath.Join("templates", "username.html"), nil)
}

func generateWaitingPage(wpd *waitingPageData) ([]byte, error) {
	return generateTemplate(filepath.Join("templates", "waiting-page.html"), wpd)
}

func generatePlayerList(pld *playerListData) ([]byte, error) {
	return generateTemplate(filepath.Join("templates", "player-list.html"), pld)
}

func generateGamePage(gpd *gamePageData) ([]byte, error) {
	return generateTemplate(filepath.Join("templates", "game-page.html"), gpd)
}

func generateVotingPage(vpd *votingPageData) ([]byte, error) {
	return generateTemplate(filepath.Join("templates", "voting-page.html"), vpd)
}

func generatePicturePreview(ipd *imagePreviewData) ([]byte, error) {
	return generateTemplate(filepath.Join("templates", "image-preview.html"), ipd)
}

func generateLeaderboardPage(lpd *leaderboardPageData) ([]byte, error) {
	return generateTemplate(filepath.Join("templates", "leaderboard.html"), lpd)
}
