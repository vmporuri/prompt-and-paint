package game

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var (
	ctx = context.Background()
	rdb = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
	})
)

func subscribeRoom(room *Room) {
	room.Pubsub = rdb.Subscribe(ctx, room.ID)
	go room.readPump()
}

func subscribeClient(client *Client) {
	client.Pubsub = rdb.Subscribe(ctx, client.RoomID)
	go client.readPump()
}

func publishClientMessage(client *Client, msg string) {
	err := rdb.Publish(ctx, client.RoomID, msg).Err()
	if err != nil {
		log.Println("Could not publish client message")
	}
}

func publishRoomMessage(room *Room, msg []byte) {
	err := rdb.Publish(ctx, room.ID, msg).Err()
	if err != nil {
		log.Println("Could not publish room message")
	}
}
