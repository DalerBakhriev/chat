package config

import (
	"log"

	"github.com/go-redis/redis/v8"
)

// Redis client
var Redis *redis.Client

// CreateRedisClient creates redis client
func CreateRedisClient() {
	opt, err := redis.ParseURL("redis://localhost:6364/0")
	if err != nil {
		log.Fatal(err)
	}

	redis := redis.NewClient(opt)

	Redis = redis
}
