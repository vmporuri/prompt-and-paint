package game

// A type that represents the current state of a player or room.
// Used to annotate database backups.
type gameState string

const (
	isReady     gameState = "is-ready"     // Player is ready
	isNotReady  gameState = "is-not-ready" // Player is not ready
	picture     gameState = "picture"      // A picture URL
	username    gameState = "username"     // A player username
	roomList    gameState = "room-list"    // The global list of all rooms
	roomID      gameState = "room-id"      // The id of a room
	leaderboard gameState = "leaderboard"  // The leaderboard for a room
	roomBackup  gameState = "room-backup"  // A room's backup
)
