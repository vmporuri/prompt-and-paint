package game

type waitingPageData struct {
	RoomID string
}

type playerListData struct {
	Players map[string]string
}

type gamePageData struct {
	Question string
}

type votingPageData struct {
	URLs []string
}

type imagePreviewData struct {
	URL string
}

type leaderboardPageData struct {
	Scores      map[string]int
	Leaderboard map[string]int
}
