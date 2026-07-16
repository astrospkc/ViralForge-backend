package utils

import (
	"context"
	"fmt"
	"viralforge/src/connect"
	"viralforge/src/models"
)

func DeleteVideoTask(ctx context.Context, videoId int64, userId int64) error {
    
    

    // Only fetch + delete upload record if it exists
    
        var uploadData models.VideoUpload
        
        err:= connect.Db.NewSelect().
            Model(&uploadData).
            Where("id = ?", videoId).
            Scan(ctx)
        if err != nil {
            return fmt.Errorf("fetching video upload: %w", err)
        }

        if _, err = DeleteFromS3(uploadData.FileURL); err != nil {
            return fmt.Errorf("deleting raw video from S3: %w", err)
        }

        _,err = connect.Db.NewUpdate().
            Model((*models.VideoUpload)(nil)).
            Set("is_deleted = ?", true).
            Set("transcode_status = ?", false).
            Set("publish_status = ?", models.PublishEnum_Draft).
            Where("id = ?", videoId).
            Exec(ctx)
        if err != nil {
            return fmt.Errorf("deleting video upload from db: %w", err)
        }
    

    // Always attempt quality cleanup (handles orphans too)
    var qualityData []models.VideoQuality
    err = connect.Db.NewSelect().
        Model(&qualityData).
        Where("video_upload_id = ?", videoId).
        Scan(ctx)
    if err != nil {
        return fmt.Errorf("fetching video qualities: %w", err)
    }

    if len(qualityData) == 0  {
        return nil // Truly nothing existed
    }

    for _, q := range qualityData {
        if _, err = DeleteFromS3(q.PlaylistKey); err != nil {
            return fmt.Errorf("deleting quality %d from S3: %w", q.ID, err)
        }
    }

    _,err = connect.Db.NewUpdate().
        Model((*models.VideoQuality)(nil)).
        Set("is_deleted = ?", true).
        Where("video_upload_id = ?", videoId).
        Exec(ctx)
    if err != nil {
        return fmt.Errorf("update is_deleted  true in video qualities: %w", err)
    }

    return nil
}