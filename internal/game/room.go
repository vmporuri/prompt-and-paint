package game

import (
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
	Mutex      *sync.RWMutex
	Pubsub     *redis.PubSub
}

const roomList string = "roomList"

func createRoom() *Room {
	room := &Room{
		ID:         shortuuid.New(),
		Players:    make(map[string]string),
		ReadyCount: 0,
		Mutex:      &sync.RWMutex{},
	}
	addToRedisSet(roomList, room.ID)
	subscribeRoom(room)
	return room
}

func (r *Room) readPump() {
	defer r.Pubsub.Close()

	ch := r.Pubsub.Channel()

	for msg := range ch {
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
		gpd := &gamePageData{Question: generateQuestion()}
		gamePage, err := json.Marshal(newPSMessage(enterGame, string(generateGamePage(gpd))))
		if err != nil {
			log.Println("Could not marshal game page")
			return
		}
		publishRoomMessage(r, gamePage)
	}
}
