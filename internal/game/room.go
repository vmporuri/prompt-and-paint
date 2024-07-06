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

type Room struct {
	ID             string
	Players        map[string]string
	PlayerStatuses map[string]bool
	ReadyCount     int
	Pubsub         *redis.PubSub
	Mutex          *sync.RWMutex
	Ctx            context.Context
	Cancel         context.CancelFunc
}

const roomList string = "roomList"

func createRoom() (*Room, error) {
	ctx, cancel := context.WithCancel(context.Background())
	room := &Room{
		ID:             shortuuid.New(),
		Players:        make(map[string]string),
		PlayerStatuses: make(map[string]bool),
		ReadyCount:     0,
		Mutex:          &sync.RWMutex{},
		Ctx:            ctx,
		Cancel:         cancel,
	}
	err := addToRedisSet(ctx, roomList, room.ID)
	if err != nil {
		log.Printf("Error adding room to room list: %v", err)
		return nil, err
	}
	room.resetReadyCount()
	subscribeRoom(room)
	return room, nil
}

func (r *Room) getPlayers() map[string]string {
	return r.Players
}

func (r *Room) addPlayerToRoom(userID, username string) {
	r.Mutex.Lock()
	r.Players[userID] = username
	r.PlayerStatuses[userID] = false
	r.Mutex.Unlock()
}

func (r *Room) deletePlayerFromRoom(userID string) {
	r.Mutex.Lock()
	delete(r.Players, userID)
	delete(r.PlayerStatuses, userID)
	r.Mutex.Unlock()
}

func (r *Room) getLeaderboardKey() string {
	return fmt.Sprintf("%s:%s", r.ID, leaderboard)
}

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

func (r *Room) addPlayerToLeaderboard(userID string) error {
	return addToRedisSortedSet(r.Ctx, r.getLeaderboardKey(), userID)
}

func (r *Room) updatePlayerScore(userID string, score int) error {
	return updateRedisSortedSet(r.Ctx, r.getLeaderboardKey(), userID, score)
}

func (r *Room) deletePlayerFromLeaderboard(userID string) error {
	return deleteFromRedisSortedSet(r.Ctx, r.getLeaderboardKey(), userID)
}

func (r *Room) getPlayerCount() int {
	return len(r.Players)
}

func (r *Room) getReadyCount() int {
	return r.ReadyCount
}

func (r *Room) updateReadyCount(userID string) error {
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

func (r *Room) resetReadyCount() {
	r.Mutex.Lock()
	r.ReadyCount = 0
	for player := range r.PlayerStatuses {
		r.PlayerStatuses[player] = false
	}
	r.Mutex.Unlock()
}

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
			case userReady:
				go r.handleReadySignal(psEvent.Msg)
			case getPicture:
				go r.handleUserSubmission(psEvent.Sender)
			case vote:
				go r.handleVote(psEvent.Sender, psEvent.Msg)
			case CloseWS:
				go r.disconnectUser(psEvent.Msg)
			}
		case <-r.Ctx.Done():
			return
		}
	}
}

func (r *Room) addUser(userID, username string) {
	err := r.addPlayerToLeaderboard(userID)
	if err != nil {
		log.Printf("Error adding user to leaderboard: %v", err)
		return
	}
	r.addPlayerToRoom(userID, username)
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

func (r *Room) handleReadySignal(userID string) {
	err := r.updateReadyCount(userID)
	if err != nil {
		log.Printf("Error updating ready count: %v", err)
		return
	}

	playerCount := r.getPlayerCount()
	readyCount := r.getReadyCount()
	if readyCount == 0 || readyCount < playerCount {
		return
	}

	r.resetReadyCount()
	gpd := &gamePageData{Question: generateQuestion(r)}
	gamePageBytes, err := generateGamePage(gpd)
	if err != nil {
		log.Printf("Error creating game page template: %v", err)
		return
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
}

func (r *Room) disconnectUser(userID string) {
	err := r.deletePlayerFromLeaderboard(userID)
	if err != nil {
		log.Printf("Error removing player from database: %v", err)
	}

	if r.getPlayerCount() == 0 {
		err := deleteFromRedisSet(r.Ctx, roomList, r.ID)
		if err != nil {
			log.Printf("Error deleting room from roomList: %v", err)
		}
		r.Cancel()
	}
}

func (r *Room) handleUserSubmission(userID string) {
	err := r.updateReadyCount(userID)
	if err != nil {
		log.Printf("Error updating ready count: %v", err)
		return
	}
	r.sendVotingPage()
}

func (r *Room) sendVotingPage() {
	playerCount := r.getPlayerCount()
	readyCount := r.getReadyCount()
	if readyCount == 0 || readyCount < playerCount {
		return
	}

	r.resetReadyCount()
	answers := make([]string, 0, playerCount)
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
}

func (r *Room) handleVote(userID, voteURL string) {
	err := r.updateReadyCount(userID)
	if err != nil {
		log.Printf("Error updating ready count: %v", err)
		return
	}
	err = incrRedisKey(r.Ctx, voteURL)
	if err != nil {
		log.Printf("Error updating vote counts: %v", err)
		return
	}
	r.countVotes()
}

func (r *Room) countVotes() {
	playerCount := r.getPlayerCount()
	readyCount := r.getReadyCount()
	if readyCount == 0 || readyCount < playerCount {
		return
	}

	r.resetReadyCount()
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

func (r *Room) sendLeaderboard(scores map[string]int, lb map[string]int) {
	lpd := &leaderboardPageData{Scores: scores, Leaderboard: lb}
	leaderboardPageBytes, err := generateLeaderboardPage(lpd)
	if err != nil {
		log.Printf("Error creating leaderboard page template: %v", err)
		return
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
}
