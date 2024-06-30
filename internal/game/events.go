package game

type gameEvent string

const (
	create        gameEvent = "create-room"
	join          gameEvent = "join-room"
	username      gameEvent = "set-username"
	ready         gameEvent = "ready"
	newPlayerList gameEvent = "new-player-list"
	enterGame     gameEvent = "game-room"
	newUser       gameEvent = "new-user"
	userReady     gameEvent = "ready"
	prompt        gameEvent = "prompt"
	votePage      gameEvent = "vote-page"
)

const CloseWS gameEvent = "close-ws"
