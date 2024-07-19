package main

import (
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Creates a Redis database connection.
func createDBConnection(cfg *Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.Database.RedisHost, cfg.Database.RedisPort),
	})
}
