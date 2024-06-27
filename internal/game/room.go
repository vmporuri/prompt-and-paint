package game

import (
	"log"
	"strings"
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

const (
	newUser   string = "new-user"
	userReady string = "ready"
)

var roomList = make(map[string]*Room, 0)

func createRoom() *Room {
	room := &Room{
		ID:         shortuuid.New(),
		Players:    make(map[string]string),
		ReadyCount: 0,
		Mutex:      &sync.RWMutex{},
	}
	roomList[room.ID] = room
	subscribeRoom(room)
	return room
}

func (r *Room) readPump() {
	defer r.Pubsub.Close()

	ch := r.Pubsub.Channel()

	for msg := range ch {
		log.Println(msg.Channel, msg.Payload)
		payload := strings.Split(msg.Payload, ":")
		event := payload[0]
		msg := payload[1]

		switch event {
		case newUser:
			r.addUser(msg)
		case userReady:
			r.updateReadyCount()
		}
	}
}

func (r *Room) addUser(msg string) {
	user := strings.Split(msg, "-")
	r.Mutex.Lock()
	r.Players[user[0]] = user[1]
	r.Mutex.Unlock()
	publishRoomMessage(r, generatePlayerList(r))
}

func (r *Room) updateReadyCount() {
	r.Mutex.Lock()
	r.ReadyCount++
	r.Mutex.Unlock()
	if r.ReadyCount > 0 && r.ReadyCount == len(r.Players) {
		publishRoomMessage(r, generateGamePage(r))
	}
}
