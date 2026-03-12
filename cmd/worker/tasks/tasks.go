// job definition  + enqueu function
package tasks

import (
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

// task name :route name for jobs
const TypeTranscodeVideo = "transcode:video"

const TypeThumbnail ="thumbnail:video"

// payload: everything the worker needs
type TranscodeVideoPayload struct{
	VideoUploadID int64 
	InputKey      string
	UserId        int64
}


func NewTranscodeVideoTask(videoUploadId int64, inputKey string, userId int64)(*asynq.Task, error){
	payload, err := json.Marshal(TranscodeVideoPayload{
		VideoUploadID: videoUploadId,
		InputKey: inputKey,
		UserId: userId,
	})

	if err!=nil{
		return nil, err
	}

	return asynq.NewTask(
		TypeTranscodeVideo,
		payload,
		asynq.MaxRetry(3),
		asynq.Timeout(30*time.Minute),
		asynq.Queue("transcoding"),
	), nil
}