package worker

import (
	"log"

	"viralforge/cmd/worker/tasks"
	"viralforge/src/env"

	"github.com/hibiken/asynq"
)


func StartWorkerServer() {
	envs:= env.NewEnv()
	redisAddr:= envs.AIVEN_SERVICE_URI
    
	redisOpt := asynq.RedisClientOpt{Addr: redisAddr}
    

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