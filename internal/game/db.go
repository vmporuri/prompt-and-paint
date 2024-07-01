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

func setRedisKey(ctx context.Context, key string, value any) error {
	return rdb.Set(ctx, key, value, time.Hour).Err()
}

func getRedisKey(ctx context.Context, key string) (string, error) {
	return rdb.Get(ctx, key).Result()
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

func deleteFromRedisSet(ctx context.Context, key string, member any) error {
	return rdb.SRem(ctx, key, member).Err()
}

func setRedisHash(ctx context.Context, hash, key string, value any) error {
	return rdb.HSet(ctx, hash, key, value).Err()
}

func getRedisHash(ctx context.Context, hash, key string) (string, error) {
	return rdb.HGet(ctx, hash, key).Result()
}

func incrRedisKey(ctx context.Context, key string) error {
	return rdb.Incr(ctx, key).Err()
}

func addToRedisSortedSet(ctx context.Context, key, member string) error {
	z := redis.Z{Score: 0, Member: member}
	return rdb.ZAdd(ctx, key, z).Err()
}

func updateRedisSortedSet(ctx context.Context, key, member string, score int) error {
	return rdb.ZIncrBy(ctx, key, float64(score), member).Err()
}

func getRedisSortedSet(ctx context.Context, key string) (map[string]int, error) {
	setSlice, err := rdb.ZRevRangeWithScores(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	set := make(map[string]int, len(setSlice))
	for _, z := range setSlice {
		member, ok := z.Member.(string)
		if !ok {
			log.Printf("Error fetching member from sorted set: %v", err)
			continue
		}
		set[member] = int(z.Score)
	}
	return set, nil
}
