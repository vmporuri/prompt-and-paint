package game

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
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

func (r *Room) readPump() {
	defer r.Pubsub.Close()

	ch := r.Pubsub.Channel()

	for {
		select {
		case msg := <-ch:
			psEvent := PSMessage{}
			err := json.Unmarshal([]byte(msg.Payload), &psEvent)
			if err != nil {
				log.Println("Could not unmarshal pubsub message")
			}

			switch psEvent.Event {
			case newUser:
				go r.addUser(psEvent.Msg, psEvent.OptMsg)
			case userReady:
				go r.updateReadyCount()
			case prompt:
				go r.handleUserSubmission()
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
	playerList, err := json.Marshal(newPSMessage(newPlayerList, string(generatePlayerList(r))))
	if err != nil {
		log.Println("Could not marshal new player list")
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
		gamePage, err := json.Marshal(newPSMessage(enterGame, string(generateGamePage(gpd))))
		if err != nil {
			log.Println("Could not marshal game page")
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
			ans, err := getRedisHash(r.Ctx, player, string(prompt))
			if err != nil {
				log.Printf("Error fetching player answer: %v", err)
				continue
			}
			answers = append(answers, ans)
		}
		rand.Shuffle(len(answers), func(i, j int) {
			answers[i], answers[j] = answers[j], answers[i]
		})

		apd := &votingPageData{URLs: answers}
		votingPage, err := json.Marshal(newPSMessage(enterGame, string(generateVotingPage(apd))))
		if err != nil {
			log.Printf("Unable to marshal voting page data: %v", err)
			return
		}
		publishRoomMessage(r, votingPage)
	}
}
