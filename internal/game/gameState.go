package game

type gameState string

const (
	isReady     gameState = "is-ready"
	isNotReady  gameState = "is-not-ready"
	picture     gameState = "picture"
	username    gameState = "username"
	roomList    gameState = "room-list"
	roomID      gameState = "room-id"
	leaderboard gameState = "leaderboard"
	roomBackup  gameState = "room-backup"
)
