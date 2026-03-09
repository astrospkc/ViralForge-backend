package worker

import (
	"log"

	"github.com/hibiken/asynq"
)
func StartWorkerServer() {
    // connect to same Redis
    redisOpt := asynq.RedisClientOpt{Addr: "localhost:6379"}

    server := asynq.NewServer(redisOpt, asynq.Config{
        Concurrency: 5,   // 5 jobs processed simultaneously
                          // tune this based on your CPU
        Queues: map[string]int{
            "transcoding": 10,  // queue name → priority weight
        },
    })

    // register which function handles which task type
    mux := asynq.NewServeMux()
    mux.HandleFunc(tasks.TypeTranscodeVideo, tasks.HandleTranscodeVideoTask)

    if err := server.Run(mux); err != nil {
        log.Fatal("worker server failed:", err)
    }
}