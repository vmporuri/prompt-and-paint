package game

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type PSMessage struct {
	Event  gameEvent `json:"event"`
	Msg    string    `json:"msg"`
	OptMsg string    `json:"optMsg"`
}

var rdb *redis.Client

func newPSMessage(event gameEvent, msg string) *PSMessage {
	return &PSMessage{
		Event: event,
		Msg:   msg,
	}
}

func newPSMessageWithOptMsg(event gameEvent, msg, optMsg string) *PSMessage {
	return &PSMessage{
		Event:  event,
		Msg:    msg,
		OptMsg: optMsg,
	}
}

func SetupDBConnection(redisConn *redis.Client) {
	rdb = redisConn
}

func subscribeRoom(room *Room) {
	room.Pubsub = rdb.Subscribe(room.Ctx, room.ID)
	go room.readPump()
}

func subscribeClient(client *Client) {
	client.Pubsub = rdb.Subscribe(client.Ctx, client.RoomID)
	go client.readPump()
}

func publishClientMessage(client *Client, msg []byte) {
	err := rdb.Publish(client.Ctx, client.RoomID, msg).Err()
	if err != nil {
		log.Println("Could not publish client message")
	}
}

func publishRoomMessage(room *Room, msg []byte) {
	err := rdb.Publish(room.Ctx, room.ID, msg).Err()
	if err != nil {
		log.Println("Could not publish room message")
	}
}

func setRedisKey(ctx context.Context, key string, value any) {
	err := rdb.Set(ctx, key, value, time.Hour).Err()
	if err != nil {
		log.Println("Could not store key")
	}
}

func addToRedisSet(ctx context.Context, key string, member any) {
	err := rdb.SAdd(ctx, key, member).Err()
	if err != nil {
		log.Println("Could not add to set")
	}
}

func checkMembershipRedisSet(ctx context.Context, key string, member any) bool {
	isMember, err := rdb.SIsMember(ctx, key, member).Result()
	if err != nil {
		log.Println("Could not check set membership")
	}
	return isMember
}

func setRedisHash(ctx context.Context, hash, key string, value any) error {
	return rdb.HSet(ctx, hash, key, value).Err()
}

func getRedisHash(ctx context.Context, hash, key string) (string, error) {
	return rdb.HGet(ctx, hash, key).Result()
}
