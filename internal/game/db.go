package game

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ctx = context.Background()
	rdb *redis.Client
)

func SetupDBConnection(redisConn *redis.Client) {
	rdb = redisConn
}

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

func setRedisKey(key string, value any) {
	err := rdb.Set(ctx, key, value, time.Hour).Err()
	if err != nil {
		log.Println("Could not store key")
	}
}

func addToRedisSet(key string, member any) {
	err := rdb.SAdd(ctx, key, member).Err()
	if err != nil {
		log.Println("Could not add to set")
	}
}

func checkMembershipRedisSet(key string, member any) bool {
	isMember, err := rdb.SIsMember(ctx, key, member).Result()
	if err != nil {
		log.Println("Could not check set membership")
	}
	return isMember
}
