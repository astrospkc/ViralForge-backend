package connect

import (
	"context"
	"crypto/tls"
	"log"
	"net/url"
	"viralforge/src/env"

	"github.com/redis/go-redis/v9"
)

var otpRedis *redis.Client

func NewOTPClient() *redis.Client {
	if otpRedis != nil {
		return otpRedis
	}

	envs := env.NewEnv()

	u, err := url.Parse(envs.AIVEN_SERVICE_URI)
	if err != nil {
		log.Fatalf("invalid AIVEN_SERVICE_URI: %v", err)
	}

	otpRedis = redis.NewClient(&redis.Options{
		Addr:      u.Host,
		Password:  envs.AIVEN_PASSWORD,
		DB:        1,
		TLSConfig: &tls.Config{},
	})
	if err := otpRedis.Ping(context.Background()).Err(); err != nil {
		log.Printf("Redis OTP ping warning: %v", err)
	}
	return otpRedis
}
