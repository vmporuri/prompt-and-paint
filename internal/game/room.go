package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"

	"github.com/lithammer/shortuuid"
	"github.com/redis/go-redis/v9"
)

// A type that represents the current room state.
type roomState string

const (
	waiting roomState = "waiting"
	playing roomState = "playing"
	voting  roomState = "voting"
	scoring roomState = "scoring"
)

// Represents a room of players, which conducts a match.
// Used to store data for the match and synchronize the game events for the players.
// Uniquely identified by RoomID.
// Communicates with players over a pub/sub channel.
type Room struct {
	ID             string
	Players        map[string]string
	PlayerStatuses map[string]bool
	State          roomState
	ReadyCount     int
	Pubsub         *redis.PubSub
	Mutex          *sync.RWMutex
	Ctx            context.Context
	Cancel         context.CancelFunc
}

// Creates a brand new room.
func createRoom() (*Room, error) {
	ctx, cancel := context.WithCancel(context.Background())
	room := &Room{
		ID:             shortuuid.New(),
		Players:        make(map[string]string),
		PlayerStatuses: make(map[string]bool),
		State:          waiting,
		ReadyCount:     0,
		Mutex:          &sync.RWMutex{},
		Ctx:            ctx,
		Cancel:         cancel,
	}
	err := roomRepo.addRoom(ctx, room.ID)
	if err != nil {
		log.Printf("Error adding room to room list: %v", err)
		return nil, err
	}
	room.resetReadyCount()
	go func() {
		_, err := room.generateQuestion()
		if err != nil {
			log.Printf("Error generating question: %v", err)
		}
	}()
	subscribeRoom(room)
	return room, nil
}

// Deletes the room. Used when all players have left the room.
// Deletes all data associated with the room in the database and stops the
// pub/sub read loop.
func (r *Room) deleteRoom() {
	err := deleteRedisKey(r.Ctx, r.ID)
	if err != nil {
		log.Printf("Error deleting room backup: %v", err)
	}
	err = deleteRedisKey(r.Ctx, r.getLeaderboardKey())
	if err != nil {
		log.Printf("Error deleting leaderboard: %v", err)
	}
	err = roomRepo.deleteRoom(r.Ctx, r.ID)
	if err != nil {
		log.Printf("Error deleting room from roomList: %v", err)
	}
	r.Cancel()
}

// Returns a map of userIDs to usernames for players connected to the room.
func (r *Room) getPlayers() map[string]string {
	return r.Players
}

// Adds a new user to the room.
func (r *Room) addPlayerToRoom(userID, username string) {
	r.Mutex.Lock()
	r.Players[userID] = username
	r.PlayerStatuses[userID] = false
	r.Mutex.Unlock()
}

// Deletes a player from the room.
func (r *Room) deletePlayerFromRoom(userID string) {
	r.Mutex.Lock()
	delete(r.Players, userID)
	delete(r.PlayerStatuses, userID)
	r.Mutex.Unlock()

	playerState, err := getRedisHash(r.Ctx, userID, string(ready))
	if err != nil {
		log.Printf("Error fetching player status: %v", err)
	}
	if playerState == string(isReady) {
		r.Mutex.Lock()
		r.ReadyCount--
		r.Mutex.Unlock()
	}
}

// Gets the key for the leaderboard stored in the database.
func (r *Room) getLeaderboardKey() string {
	return fmt.Sprintf("%s:%s", r.ID, leaderboard)
}

// Retrieves the leaderboard from the database.
func (r *Room) getLeaderboard() (map[string]int, error) {
	players := r.getPlayers()
	lbWithIDs, err := getRedisSortedSetWithScores(r.Ctx, r.getLeaderboardKey())
	if err != nil {
		return nil, err
	}
	lb := make(map[string]int, len(lbWithIDs))
	for userID := range lbWithIDs {
		username, ok := players[userID]
		if !ok {
			log.Printf("Error unknown player on leaderboard: %s", userID)
			continue
		}
		lb[username] = lbWithIDs[userID]
	}
	return lb, nil
}

// Adds a new player to the leaderboard if they haven't previously joined the game.
func (r *Room) addPlayerToLeaderboard(userID string) error {
	alreadyExists, err := checkMembershipRedisSortedSet(r.Ctx, r.getLeaderboardKey(), userID)
	if err != nil {
		log.Printf("Error checking if user is already on leaderboard: %v", err)
	} else if alreadyExists {
		return nil
	}
	return addToRedisSortedSet(r.Ctx, r.getLeaderboardKey(), userID)
}

// Updates the (total) score for the player with id userID by their round score.
func (r *Room) updatePlayerScore(userID string, score int) error {
	return updateRedisSortedSet(r.Ctx, r.getLeaderboardKey(), userID, score)
}

// Deletes a player from the leaderboard.
func (r *Room) deletePlayerFromLeaderboard(userID string) error {
	return deleteFromRedisSortedSet(r.Ctx, r.getLeaderboardKey(), userID)
}

// Gets the total number of players.
func (r *Room) getPlayerCount() int {
	return len(r.Players)
}

// Gets the number of unique players who are ready for the next game event.
func (r *Room) getReadyCount() int {
	return r.ReadyCount
}

// Increments the ready count if the userID has not already been marked as ready.
func (r *Room) incrReadyCount(userID string) error {
	status, err := getRedisHash(r.Ctx, userID, string(ready))
	if err != nil {
		return err
	} else if status != string(isReady) {
		return errors.New("Player is not ready")
	} else if r.PlayerStatuses[userID] {
		return errors.New("Player is already marked as ready")
	}

	r.Mutex.Lock()
	r.ReadyCount++
	r.PlayerStatuses[userID] = true
	r.Mutex.Unlock()
	return nil
}

// Resets the ready count back to zero.
func (r *Room) resetReadyCount() {
	r.Mutex.Lock()
	r.ReadyCount = 0
	for player := range r.PlayerStatuses {
		r.PlayerStatuses[player] = false
	}
	r.Mutex.Unlock()
}

// Retrieves the current question for the round.
func (r *Room) getQuestion() (string, error) {
	return getRedisHash(r.Ctx, r.ID, "question")
}

// Generates a new question for the next round.
// Makes an OpenAI request, so should be called in a separate goroutine.
func (r *Room) generateQuestion() (string, error) {
	question, err := generateAIQuestion(r.Ctx)
	if err != nil {
		return "", err
	}
	return question, setRedisHash(r.Ctx, r.ID, "question", question)
}

// Backs up the room state to the database.
// Used when a player reconnects to the room.
func (r *Room) backupRoomState(template []byte) error {
	return setRedisHash(r.Ctx, r.ID, string(roomBackup), template)
}

// Updates the room state to match the current game event.
func (r *Room) updateRoomState(newState roomState) {
	r.Mutex.Lock()
	r.State = newState
	r.Mutex.Unlock()
}

// Reads client events from the pub/sub channel. Dispatches the appropriate event handler.
func (r *Room) readPump() {
	defer r.Pubsub.Close()

	ch := r.Pubsub.Channel()

	for {
		select {
		case msg := <-ch:
			psEvent := PSMessage{}
			err := json.Unmarshal([]byte(msg.Payload), &psEvent)
			if err != nil {
				log.Printf("Error unmarshalling pubsub message: %v", err)
				continue
			}

			switch psEvent.Event {
			case newUser:
				go r.addUser(psEvent.Sender, psEvent.Msg)
			case reconnect:
				go r.connectUser(psEvent.Sender, psEvent.Msg)
			case ready:
				go r.handleReadySignal(psEvent.Msg)
			case getPicture:
				go r.handleUserSubmission(psEvent.Sender)
			case vote:
				go r.handleVote(psEvent.Sender, psEvent.Msg)
			case leave, CloseWS:
				go r.disconnectUser(psEvent.Msg)
			}
		case <-r.Ctx.Done():
			return
		}
	}
}

// Checks if all players are ready. If so, triggers appropriate update function.
func (r *Room) checkRoomState() {
	playerCount := r.getPlayerCount()
	readyCount := r.getReadyCount()
	if readyCount == 0 || readyCount < playerCount {
		return
	}
	r.resetReadyCount()

	switch r.State {
	case waiting, scoring:
		r.sendGamePage()
	case playing:
		r.sendVotingPage()
	case voting:
		r.countVotes()
	}
}

// Updates the room's internal state with the provided user information.
func (r *Room) connectUser(userID, username string) {
	err := r.addPlayerToLeaderboard(userID)
	if err != nil {
		log.Printf("Error adding user to leaderboard: %v", err)
		return
	}
	r.addPlayerToRoom(userID, username)
}

// Connects user and publishes the updated list of players.
func (r *Room) addUser(userID, username string) {
	r.connectUser(userID, username)
	players := r.getPlayers()
	pld := &playerListData{Players: players}
	playerListBytes, err := generatePlayerList(pld)
	if err != nil {
		log.Printf("Error creating player list template: %v", err)
		r.deletePlayerFromRoom(userID)
		err := r.deletePlayerFromLeaderboard(userID)
		if err != nil {
			log.Printf("Error removing player from database: %v", err)
		}
		return
	}
	playerList, err := json.Marshal(newPSMessage(newPlayerList, r.ID, string(playerListBytes)))
	if err != nil {
		log.Printf("Error marshalling new player list: %v", err)
		r.deletePlayerFromRoom(userID)
		err := r.deletePlayerFromLeaderboard(userID)
		if err != nil {
			log.Printf("Error removing player from database: %v", err)
		}
		return
	}
	err = publishRoomMessage(r, playerList)
	if err != nil {
		log.Printf("Error publishing new player list: %v", err)
		r.deletePlayerFromRoom(userID)
		err := r.deletePlayerFromLeaderboard(userID)
		if err != nil {
			log.Printf("Error removing player from database: %v", err)
		}
		return
	}
}

// Handles a ready signal from a client.
// If enough players are ready, the room state is updated to the next game event.
func (r *Room) handleReadySignal(userID string) {
	err := r.incrReadyCount(userID)
	if err != nil {
		log.Printf("Error updating ready count: %v", err)
		return
	}
	r.checkRoomState()
}

// Sends the HTML for the game page to all clients via the pub/sub channel.
func (r *Room) sendGamePage() {
	question, err := r.getQuestion()
	if err != nil {
		question, err = r.generateQuestion()
		if err != nil {
			log.Printf("Could not generate a prompt: %v", err)
			return
		}
	}
	gpd := &gamePageData{Question: question}
	go func() {
		_, err := r.generateQuestion()
		if err != nil {
			log.Printf("Error generating question: %v", err)
		}
	}()
	gamePageBytes, err := generateGamePage(gpd)
	if err != nil {
		log.Printf("Error creating game page template: %v", err)
		return
	}
	err = r.backupRoomState(gamePageBytes)
	if err != nil {
		log.Printf("Error backing up game page data: %v", err)
	}
	gamePage, err := json.Marshal(newPSMessage(enterGame, r.ID, string(gamePageBytes)))
	if err != nil {
		log.Printf("Error marshalling game page: %v", err)
		return
	}
	err = publishRoomMessage(r, gamePage)
	if err != nil {
		log.Printf("Error publishing game page: %v", err)
		return
	}
	r.updateRoomState(playing)
}

// Handles user disconnection.
// Agnostic to whether the disconnection was user-initiated or unexpected.
func (r *Room) disconnectUser(userID string) {
	r.deletePlayerFromRoom(userID)

	if r.getPlayerCount() == 0 {
		r.deleteRoom()
	}
	r.checkRoomState()
}

// Handles a user submitted picture by updating the ready count.
func (r *Room) handleUserSubmission(userID string) {
	err := r.incrReadyCount(userID)
	if err != nil {
		log.Printf("Error updating ready count: %v", err)
		return
	}
	r.checkRoomState()
}

// Sends the HTML for the voting page to all clients via the pub/sub channel.
func (r *Room) sendVotingPage() {
	answers := make([]string, 0, r.getPlayerCount())
	players := r.getPlayers()
	for player := range players {
		ans, err := getRedisHash(r.Ctx, player, string(picture))
		if err != nil {
			log.Printf("Error fetching player answer: %v", err)
			continue
		}
		answers = append(answers, ans)
	}
	rand.Shuffle(len(answers), func(i, j int) {
		answers[i], answers[j] = answers[j], answers[i]
	})

	for _, ans := range answers {
		err := setRedisKey(r.Ctx, ans, 0)
		if err != nil {
			log.Printf("Error initializing vote count: %v", err)
			continue
		}
	}

	apd := &votingPageData{URLs: answers}
	votingPageBytes, err := generateVotingPage(apd)
	if err != nil {
		log.Printf("Error creating voting page template: %v", err)
		return
	}
	err = r.backupRoomState(votingPageBytes)
	if err != nil {
		log.Printf("Error backing up game page data: %v", err)
	}
	votingPage, err := json.Marshal(newPSMessage(votePage, r.ID, string(votingPageBytes)))
	if err != nil {
		log.Printf("Error marshalling voting page data: %v", err)
		return
	}
	err = publishRoomMessage(r, votingPage)
	if err != nil {
		log.Printf("Error publishing voting page: %v", err)
		return
	}
	r.updateRoomState(voting)
}

// Records a player's vote.
func (r *Room) handleVote(userID, voteURL string) {
	err := r.incrReadyCount(userID)
	if err != nil {
		log.Printf("Error updating ready count: %v", err)
		return
	}
	err = incrRedisKey(r.Ctx, voteURL)
	if err != nil {
		log.Printf("Error updating vote counts: %v", err)
		return
	}
	r.checkRoomState()
}

// Counts all the votes for each picture and updates each player's total score.
func (r *Room) countVotes() {
	scores := make(map[string]int)
	players := r.getPlayers()
	for player, username := range players {
		url, err := getRedisHash(r.Ctx, player, string(picture))
		if err != nil {
			log.Printf("Error fetching player answer: %v", err)
			continue
		}
		countString, err := getRedisKey(r.Ctx, url)
		if err != nil {
			log.Printf("Error retrieving vote count: %v", err)
			continue
		}
		count, err := strconv.Atoi(countString)
		if err != nil {
			log.Printf("Error processing count: %v", err)
			continue
		}
		scores[username] = count
	}

	for player, username := range players {
		err := r.updatePlayerScore(player, scores[username])
		if err != nil {
			log.Printf("Error updating player score: %v", err)
		}
	}

	lb, err := r.getLeaderboard()
	if err != nil {
		log.Printf("Error retrieving leaderboard: %v", err)
		return
	}

	r.sendLeaderboard(scores, lb)
}

// Sends the HTML for the leaderboard page to all clients via the pub/sub channel.
func (r *Room) sendLeaderboard(scores map[string]int, lb map[string]int) {
	lpd := &leaderboardPageData{Scores: scores, Leaderboard: lb}
	leaderboardPageBytes, err := generateLeaderboardPage(lpd)
	if err != nil {
		log.Printf("Error creating leaderboard page template: %v", err)
		return
	}
	err = r.backupRoomState(leaderboardPageBytes)
	if err != nil {
		log.Printf("Error backing up game page data: %v", err)
	}
	leaderboardPage, err := json.Marshal(
		newPSMessage(sendLeaderboard, r.ID, string(leaderboardPageBytes)),
	)
	if err != nil {
		log.Printf("Error marshalling leaderboard page: %v", err)
		return
	}
	err = publishRoomMessage(r, leaderboardPage)
	if err != nil {
		log.Printf("Error publishing leaderboard: %v", err)
		return
	}
	r.updateRoomState(scoring)
}
