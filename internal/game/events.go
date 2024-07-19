package game

// A type that represents the various types of user-driven game events.
type gameEvent string

const (
	create          gameEvent = "create-room"      // Room been created
	join            gameEvent = "join-room"        // Room been joined
	setUsername     gameEvent = "set-username"     // User set username
	newUser         gameEvent = "new-user"         // New user joined
	newPlayerList   gameEvent = "new-player-list"  // Player list updated
	ready           gameEvent = "ready"            // User is ready for next round
	enterGame       gameEvent = "game-room"        // Game started
	prompt          gameEvent = "prompt"           // User submitted prompt
	getPicture      gameEvent = "get-picture"      // Get user's chosen picture
	pickPicture     gameEvent = "pick-picture"     // User picked picture
	votePage        gameEvent = "vote-page"        // Send the page of candidates
	vote            gameEvent = "vote"             // User voted
	sendLeaderboard gameEvent = "send-leaderboard" // Send the current leaderboard
	leave           gameEvent = "leave"            // User left game
	reconnect       gameEvent = "reconnect"        // User has reconnected
	CloseWS         gameEvent = "close-ws"         // Unexpected WebSocket disconnection.
)
