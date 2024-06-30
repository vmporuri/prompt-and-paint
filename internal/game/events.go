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
)

const CloseWS gameEvent = "close-ws"
