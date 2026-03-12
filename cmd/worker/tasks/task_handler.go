// worker handler (does actualwork)
package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"viralforge/service"

	"github.com/hibiken/asynq"
)

func HandleTranscodeVideoTask(ctx context.Context, t*asynq.Task) error {
	
	// unpack payload 
	var payload TranscodeVideoPayload 
	if err := json.Unmarshal(t.Payload(), &payload); err!=nil{
		return fmt.Errorf("failed to unmarshal payload %w", err)
	}

	fmt.Println("payload : ", payload)


	// call hls transcode 
	err := service.HLSTranscode(payload.VideoUploadID, payload.InputKey, payload.UserId)
	if err!=nil{
		return err
	}
	return nil

}


