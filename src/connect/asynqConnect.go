package connect

import (
	"crypto/tls"
	"log"
	"net/url"
	"viralforge/src/env"

	"github.com/hibiken/asynq"
)

var AsynqClient *asynq.Client

func AsynqConnect() {
    envs := env.NewEnv()

    u, err := url.Parse(envs.AIVEN_SERVICE_URI)
    if err != nil {
        log.Fatal("invalid aiven URI:", err)
    }

    password:= envs.AIVEN_PASSWORD

    AsynqClient = asynq.NewClient(asynq.RedisClientOpt{
        Addr:      u.Host,
        Password:  password,
        TLSConfig: &tls.Config{},
    })

    log.Println("asynq client connected to:", u.Host, AsynqClient)
}