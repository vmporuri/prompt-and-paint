package game

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// A data structure used to format all internal pub/sub messages.
type PSMessage struct {
	Event  gameEvent `json:"event"`
	Sender string    `json:"sender"`
	Msg    string    `json:"msg"`
}

var rdb *redis.Client

const expireTime = time.Hour

// Sets up the Redis database connection.
func SetupDBConnection(redisConn *redis.Client) {
	rdb = redisConn
}

// Constructs a new PSMessage using the provided arguments.
func newPSMessage(event gameEvent, sender, msg string) *PSMessage {
	return &PSMessage{
		Event:  event,
		Sender: sender,
		Msg:    msg,
	}
}

// Subscribes a room to a pub/sub channel.
func subscribeRoom(room *Room) {
	room.Pubsub = rdb.Subscribe(room.Ctx, room.ID)
	go room.readPump()
}

// Subscribes a client to a pub/sub channel.
func subscribeClient(client *Client) {
	client.Pubsub = rdb.Subscribe(client.Ctx, client.RoomID)
	go client.readPump()
}

// Publishes a client message to their subscribed channel.
func publishClientMessage(client *Client, msg []byte) error {
	return rdb.Publish(client.Ctx, client.RoomID, msg).Err()
}

// Publishes a room message to their subscribed channel.
func publishRoomMessage(room *Room, msg []byte) error {
	return rdb.Publish(room.Ctx, room.ID, msg).Err()
}

// Sets a key in database.
// Errors if database query errors.
func setRedisKey(ctx context.Context, key string, value any) error {
	err := rdb.Set(ctx, key, value, expireTime).Err()
	if err != nil {
		return err
	}
	return rdb.Expire(ctx, key, expireTime).Err()
}

// Gets the value associated with a key in database.
// Errors if database query errors.
func getRedisKey(ctx context.Context, key string) (string, error) {
	return rdb.Get(ctx, key).Result()
}

// Increments the value associated with a key.
// Errors if the database query errors.
func incrRedisKey(ctx context.Context, key string) error {
	return rdb.Incr(ctx, key).Err()
}

// Adds to a specified set in database. Creates the set if it does not yet exist.
// Errors if database query errors.
func addToRedisSet(ctx context.Context, key string, member any) error {
	err := rdb.SAdd(ctx, key, member).Err()
	if err != nil {
		return err
	}
	return rdb.Expire(ctx, key, expireTime).Err()
}

// Checks membership to a set in database.
// Errors if database query errors.
func checkMembershipRedisSet(ctx context.Context, key string, member any) (bool, error) {
	return rdb.SIsMember(ctx, key, member).Result()
}

// Deletes from a set in database.
// Errors if database query errors.
func deleteFromRedisSet(ctx context.Context, key string, member any) error {
	return rdb.SRem(ctx, key, member).Err()
}

// Sets a hash value in database. Creates the hash if it does not yet exist.
// Errors if database query errors.
func setRedisHash(ctx context.Context, hash, key string, value any) error {
	err := rdb.HSet(ctx, hash, key, value).Err()
	if err != nil {
		return err
	}
	return rdb.Expire(ctx, hash, expireTime).Err()
}

// Gets a value associated with a hash in database.
// Errors if database query errors.
func getRedisHash(ctx context.Context, hash, key string) (string, error) {
	return rdb.HGet(ctx, hash, key).Result()
}

// Deletes hash (the whole hash, not just a key) from the database.
// Errors if the database query errors.
func deleteRedisHash(ctx context.Context, hash string) error {
	return rdb.Del(ctx, hash).Err()
}

// Adds to a sorted set in the database. Creates the set if it does not exist.
// Errors if the database query errors.
func addToRedisSortedSet(ctx context.Context, key, member string) error {
	z := redis.Z{Score: 0, Member: member}
	err := rdb.ZAdd(ctx, key, z).Err()
	if err != nil {
		return err
	}
	return rdb.Expire(ctx, key, expireTime).Err()
}

// Increments the score associated with a member of a sorted set by score.
// Errors if the database query errors.
func updateRedisSortedSet(ctx context.Context, key, member string, score int) error {
	return rdb.ZIncrBy(ctx, key, float64(score), member).Err()
}

// Retrieves a sorted set with the scores as a map from members to scores.
// Errors if the database query errors.
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

// Deletes member from a sorted set.
// Errors if the database query errors.
func deleteFromRedisSortedSet(ctx context.Context, key, member string) error {
	return rdb.ZRem(ctx, key, member).Err()
}
