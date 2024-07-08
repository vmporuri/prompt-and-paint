package game

type dbField string

const (
	isReady     dbField = "is-ready"
	isNotReady  dbField = "is-not-ready"
	picture     dbField = "picture"
	username    dbField = "username"
	roomList    dbField = "room-list"
	roomID      dbField = "room-id"
	leaderboard dbField = "leaderboard"
	roomState   dbField = "room-state"
)
