package game

import (
	"log"
	"strings"

	"github.com/lithammer/shortuuid"
	"github.com/redis/go-redis/v9"
)

type Room struct {
	ID      string
	Players []string
	Pubsub  *redis.PubSub
}

const (
	newUser string = "new-user"
)

var roomList = make(map[string]*Room, 0)

func createRoom() *Room {
	room := &Room{
		ID:      shortuuid.New(),
		Players: make([]string, 0),
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
		}
	}
}

func (r *Room) addUser(username string) {
	r.Players = append(r.Players, username)
	publishRoomMessage(r, generatePlayerList(r))
}
