package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"viralforge/src/utils"

	"github.com/hibiken/asynq"
)



func HandleTranscodeVideoTask(ctx context.Context, t*asynq.Task) error {
	
	// unpack payload 
	var payload TranscodeVideoPayload 
	if err := json.Unmarshal(t.Payload(), &payload); err!=nil{
		return fmt.Errorf("failed to unmarshal payload %w", err)
	}


	// call hls transcode 
	err := utils.HLSTranscodeandThumbnail(payload.VideoUploadID, payload.InputKey, payload.UserId)
	fmt.Println("error while transcoding and thumbnail: ", err)
	if err!=nil{
		return err
	}
	return nil

}

func HandleDeleteVideoTask(ctx context.Context, t*asynq.Task) error{
	var payload DeleteVideoPayload 
	if err:= json.Unmarshal(t.Payload() , &payload); err!=nil{
		return fmt.Errorf("failed to unmarshal payload %w", err)
	}

	
	err:= utils.DeleteVideoTask( ctx, payload.VideoUploadID, payload.UserID)
	if err!=nil{
		return err
	}
	return nil
}


