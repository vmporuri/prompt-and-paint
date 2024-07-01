package game

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"math/rand"
	"strconv"
	"sync"

	"github.com/lithammer/shortuuid"
	"github.com/redis/go-redis/v9"
)

type Room struct {
	ID         string
	Players    map[string]string
	ReadyCount int
	Pubsub     *redis.PubSub
	Mutex      *sync.RWMutex
	Ctx        context.Context
	Cancel     context.CancelFunc
}

const roomList string = "roomList"

func createRoom() *Room {
	ctx, cancel := context.WithCancel(context.Background())
	room := &Room{
		ID:         shortuuid.New(),
		Players:    make(map[string]string),
		ReadyCount: 0,
		Mutex:      &sync.RWMutex{},
		Ctx:        ctx,
		Cancel:     cancel,
	}
	addToRedisSet(ctx, roomList, room.ID)
	subscribeRoom(room)
	return room
}

func (r *Room) getPlayersKey() string {
	return fmt.Sprintf("%s:players", r.ID)
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
			}

			switch psEvent.Event {
			case newUser:
				go r.addUser(psEvent.Msg, psEvent.OptMsg)
			case userReady:
				go r.updateReadyCount()
			case picture:
				go r.handleUserSubmission()
			case vote:
				go r.handleVote(psEvent.Msg)
			case CloseWS:
				go r.disconnectUser(psEvent.Msg)
			}
		case <-r.Ctx.Done():
			return
		}
	}
}

func (r *Room) addUser(userID, username string) {
	r.Mutex.Lock()
	r.Players[userID] = username
	r.Mutex.Unlock()
	err := addToRedisSortedSet(r.Ctx, r.getPlayersKey(), userID)
	if err != nil {
		log.Printf("Error adding new user: %v", err)
		delete(r.Players, userID)
		return
	}

	pld := &playerListData{Players: r.Players}
	playerListBytes, err := generatePlayerList(pld)
	if err != nil {
		log.Printf("Error creating player list template: %v", err)
		return
	}
	playerList, err := json.Marshal(newPSMessage(newPlayerList, string(playerListBytes)))
	if err != nil {
		log.Printf("Error marshalling new player list: %v", err)
		delete(r.Players, userID)
		return
	}
	publishRoomMessage(r, playerList)
}

func (r *Room) updateReadyCount() {
	r.Mutex.Lock()
	r.ReadyCount++
	if r.ReadyCount > 0 && r.ReadyCount == len(r.Players) {
		r.ReadyCount = 0
		gpd := &gamePageData{Question: generateQuestion(r)}
		gamePageBytes, err := generateGamePage(gpd)
		if err != nil {
			log.Printf("Error creating game page template: %v", err)
			return
		}
		gamePage, err := json.Marshal(newPSMessage(enterGame, string(gamePageBytes)))
		if err != nil {
			log.Printf("Error marshalling game page: %v", err)
			return
		}
		publishRoomMessage(r, gamePage)
	}
	r.Mutex.Unlock()
}

func (r *Room) disconnectUser(userID string) {
	r.Mutex.Lock()
	delete(r.Players, userID)
	if len(r.Players) == 0 {
		err := deleteFromRedisSet(r.Ctx, roomList, r.ID)
		if err != nil {
			log.Printf("Error deleting room from roomList: %v", err)
		}
		r.Cancel()
	}
	r.Mutex.Unlock()
}

func (r *Room) handleUserSubmission() {
	r.Mutex.Lock()
	r.ReadyCount++
	r.Mutex.Unlock()
	r.sendVotingPage()
}

func (r *Room) sendVotingPage() {
	if r.ReadyCount > 0 && r.ReadyCount == len(r.Players) {
		r.Mutex.Lock()
		r.ReadyCount = 0
		r.Mutex.Unlock()

		answers := make([]string, 0, len(r.Players))
		for player := range r.Players {
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
		votingPage, err := json.Marshal(newPSMessage(votePage, string(votingPageBytes)))
		if err != nil {
			log.Printf("Error marshalling voting page data: %v", err)
			return
		}
		publishRoomMessage(r, votingPage)
	}
}

func (r *Room) handleVote(voteURL string) {
	r.Mutex.Lock()
	r.ReadyCount++
	r.Mutex.Unlock()
	err := incrRedisKey(r.Ctx, voteURL)
	if err != nil {
		log.Printf("Error updating vote counts: %v", err)
		return
	}
	r.countVotes()
}

func (r *Room) countVotes() {
	if r.ReadyCount == 0 || r.ReadyCount < len(r.Players) {
		return
	}

	r.Mutex.Lock()
	r.ReadyCount = 0
	r.Mutex.Unlock()

	scores := make(map[string]int)
	for player, username := range r.Players {
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

	for player, username := range r.Players {
		err := updateRedisSortedSet(r.Ctx, r.getPlayersKey(), player, scores[username])
		if err != nil {
			log.Printf("Error updating player score: %v", err)
		}
	}

	lb, err := getRedisSortedSet(r.Ctx, r.getPlayersKey())
	if err != nil {
		log.Printf("Error retrieving leaderboard: %v", err)
	}
	for player := range maps.Clone(lb) {
		username, ok := r.Players[player]
		if !ok {
			log.Printf("Error unknown player on leaderboard: %s", player)
			continue
		}
		lb[username] = lb[player]
		delete(lb, player)
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
		newPSMessage(leaderboard, string(leaderboardPageBytes)),
	)
	if err != nil {
		log.Printf("Error marshalling leaderboard page: %v", err)
		return
	}
	publishRoomMessage(r, leaderboardPage)
}
