package game

type gameEvent string

const (
	create          gameEvent = "create-room"
	join            gameEvent = "join-room"
	reconnect       gameEvent = "reconnect"
	setUsername     gameEvent = "set-username"
	ready           gameEvent = "ready"
	newPlayerList   gameEvent = "new-player-list"
	enterGame       gameEvent = "game-room"
	newUser         gameEvent = "new-user"
	userReady       gameEvent = "ready"
	prompt          gameEvent = "prompt"
	getPicture      gameEvent = "get-picture"
	votePage        gameEvent = "vote-page"
	pickPicture     gameEvent = "pick-picture"
	vote            gameEvent = "vote"
	sendLeaderboard gameEvent = "send-leaderboard"
	leave           gameEvent = "leave"
	CloseWS         gameEvent = "close-ws"
)
