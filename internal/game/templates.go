package game

import (
	"bytes"
	"html/template"
	"path/filepath"
)

// An abstract function that creates a template given a path to that template.
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

// Creates the username page from its template.
func generateUsername() ([]byte, error) {
	return generateTemplate(filepath.Join("templates", "username.html"), nil)
}

// Creates the waiting room from its template.
func generateWaitingPage(wpd *waitingPageData) ([]byte, error) {
	return generateTemplate(filepath.Join("templates", "waiting-page.html"), wpd)
}

// Creates the player list from its template.
func generatePlayerList(pld *playerListData) ([]byte, error) {
	return generateTemplate(filepath.Join("templates", "player-list.html"), pld)
}

// Creates the game page from its template.
func generateGamePage(gpd *gamePageData) ([]byte, error) {
	return generateTemplate(filepath.Join("templates", "game-page.html"), gpd)
}

// Creates the voting page from its template.
func generateVotingPage(vpd *votingPageData) ([]byte, error) {
	return generateTemplate(filepath.Join("templates", "voting-page.html"), vpd)
}

// Creates a picture preview from its template.
func generatePicturePreview(ipd *imagePreviewData) ([]byte, error) {
	return generateTemplate(filepath.Join("templates", "image-preview.html"), ipd)
}

// Creates the leaderboard page from its template.
func generateLeaderboardPage(lpd *leaderboardPageData) ([]byte, error) {
	return generateTemplate(filepath.Join("templates", "leaderboard.html"), lpd)
}
