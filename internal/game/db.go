package game

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type PSMessage struct {
	Event  gameEvent `json:"event"`
	Sender string    `json:"sender"`
	Msg    string    `json:"msg"`
}

var rdb *redis.Client

const expireTime = time.Hour

func newPSMessage(event gameEvent, sender, msg string) *PSMessage {
	return &PSMessage{
		Event:  event,
		Sender: sender,
		Msg:    msg,
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

func publishClientMessage(client *Client, msg []byte) error {
	return rdb.Publish(client.Ctx, client.RoomID, msg).Err()
}

func publishRoomMessage(room *Room, msg []byte) error {
	return rdb.Publish(room.Ctx, room.ID, msg).Err()
}

func setRedisKey(ctx context.Context, key string, value any) error {
	err := rdb.Set(ctx, key, value, expireTime).Err()
	if err != nil {
		return err
	}
	return rdb.Expire(ctx, key, expireTime).Err()
}

func getRedisKey(ctx context.Context, key string) (string, error) {
	return rdb.Get(ctx, key).Result()
}

func addToRedisSet(ctx context.Context, key string, member any) error {
	err := rdb.SAdd(ctx, key, member).Err()
	if err != nil {
		return err
	}
	return rdb.Expire(ctx, key, expireTime).Err()
}

func checkMembershipRedisSet(ctx context.Context, key string, member any) (bool, error) {
	return rdb.SIsMember(ctx, key, member).Result()
}

func deleteFromRedisSet(ctx context.Context, key string, member any) error {
	return rdb.SRem(ctx, key, member).Err()
}

func setRedisHash(ctx context.Context, hash, key string, value any) error {
	err := rdb.HSet(ctx, hash, key, value).Err()
	if err != nil {
		return err
	}
	return rdb.Expire(ctx, hash, expireTime).Err()
}

func getRedisHash(ctx context.Context, hash, key string) (string, error) {
	return rdb.HGet(ctx, hash, key).Result()
}

func incrRedisKey(ctx context.Context, key string) error {
	return rdb.Incr(ctx, key).Err()
}

func addToRedisSortedSet(ctx context.Context, key, member string) error {
	z := redis.Z{Score: 0, Member: member}
	err := rdb.ZAdd(ctx, key, z).Err()
	if err != nil {
		return err
	}
	return rdb.Expire(ctx, key, expireTime).Err()
}

func updateRedisSortedSet(ctx context.Context, key, member string, score int) error {
	return rdb.ZIncrBy(ctx, key, float64(score), member).Err()
}

func getRedisSortedSetWithScores(ctx context.Context, key string) (map[string]int, error) {
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

func deleteFromRedisSortedSet(ctx context.Context, key, member string) error {
	return rdb.ZRem(ctx, key, member).Err()
}

func deleteRedisHash(ctx context.Context, hash string) error {
	return rdb.Del(ctx, hash).Err()
}
