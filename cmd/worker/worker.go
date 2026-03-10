package worker

import (
	"crypto/tls"
	"log"
	"net/url"

	"viralforge/cmd/worker/tasks"
	"viralforge/src/env"

	"github.com/hibiken/asynq"
)


func StartWorkerServer() {
	envs:= env.NewEnv()
    // parse the full URI
    u, err := url.Parse(envs.AIVEN_SERVICE_URI)
    if err != nil {
        log.Fatal("invalid aiven URI:", err)
    }

    // extract password
    password:= envs.AIVEN_PASSWORD

    // print to verify parsing is correct
    log.Println("connecting to:", u.Host)

    redisOpt := asynq.RedisClientOpt{
        Addr:      u.Host,          // "host:port" only
        Password:  password,
        TLSConfig: &tls.Config{},   // Aiven requires TLS (rediss://)
    }

    server := asynq.NewServer(redisOpt, asynq.Config{
        Concurrency: 5,
        Queues: map[string]int{
            "transcoding": 10,
        },
    })

    // register which function handles which task type
    mux := asynq.NewServeMux()
    mux.HandleFunc(tasks.TypeTranscodeVideo, tasks.HandleTranscodeVideoTask)

    if err := server.Run(mux); err != nil {
        log.Fatal("worker server failed:", err)
    }
}
