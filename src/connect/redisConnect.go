package connect

import (
	"viralforge/src/env"

	"github.com/redis/go-redis/v9"
)

var otpRedis *redis.Client

func NewOTPClient() *redis.Client {
	envs := env.NewEnv()
	return redis.NewClient(&redis.Options{
		Addr:     envs.AIVEN_SERVICE_URI,
		Password: envs.AIVEN_PASSWORD,
		DB:       1,
	})
}
