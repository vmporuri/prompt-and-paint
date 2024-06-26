package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lithammer/shortuuid"
	"github.com/olahol/melody"
	"github.com/redis/go-redis/v9"
)

var (
	ctx = context.Background()
	rdb = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
	})
)

func createRoom(s *melody.Session) {
	roomID := shortuuid.New()
	s.Set("roomID", roomID)
	s.Set("pubsub", rdb.Subscribe(ctx, roomID))
}
