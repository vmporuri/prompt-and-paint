package game

// Holds data needed to create the waiting page from its template.
type waitingPageData struct {
	RoomID string
}

// Holds data needed to create the player list from its template.
type playerListData struct {
	Players map[string]string
}

// Holds data needed to create the game page from its template.
type gamePageData struct {
	Question string
}

// Holds data needed to create the voting page from its template.
type votingPageData struct {
	URLs []string
}

// Holds data needed to create the image preview from its template.
type imagePreviewData struct {
	URL string
}

// Holds data needed to create the leaderboard page from its template.
type leaderboardPageData struct {
	Scores      map[string]int
	Leaderboard map[string]int
}
