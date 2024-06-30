package game

import (
	"context"
	"encoding/json"
	"log"
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
				r.addUser(psEvent.Msg, psEvent.OptMsg)
			case userReady:
				r.updateReadyCount()
			case CloseWS:
				r.disconnectUser(psEvent.Msg)
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
	r.Mutex.Unlock()
	if r.ReadyCount > 0 && r.ReadyCount == len(r.Players) {
		r.Mutex.Lock()
		r.ReadyCount = 0
		r.Mutex.Unlock()
		gpd := &gamePageData{Question: generateQuestion(r)}
		gamePage, err := json.Marshal(newPSMessage(enterGame, string(generateGamePage(gpd))))
		if err != nil {
			log.Println("Could not marshal game page")
			return
		}
		publishRoomMessage(r, gamePage)
	}
}

func (r *Room) disconnectUser(userID string) {
	r.Mutex.Lock()
	delete(r.Players, userID)
	if len(r.Players) == 0 {
		r.Cancel()
	}
	r.Mutex.Unlock()
}
