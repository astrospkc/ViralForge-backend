package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"viralforge/src/connect"
	"viralforge/src/env"
	"viralforge/src/utils"

	// "viralforge/src/handlers"

	"viralforge/src/models"

	"github.com/google/uuid"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type Quality struct {
	Name       string
	Resolution string
	Bitrate    string
	AudioRate  string  // ← add this
}
func HLSTranscode(videoUploadId int64, inputKey string, userId int64) error {

    // check if already exists
    exists, err := connect.Db.NewSelect().
        Model((*models.VideoQuality)(nil)).
        Where("video_upload_id = ?", videoUploadId).
        Exists(context.Background())
    if err != nil {
        return fmt.Errorf("failed to check if video exists: %w", err)
    }
    if exists {
        return nil
    }

    // download from S3
    inputFile, err := utils.DownloadFromS3(inputKey)
    if err != nil {
        return fmt.Errorf("error while downloading from s3: %w", err)
    }
    defer os.Remove(inputFile)

    var qualities = []Quality{
        {Name: "1080p", Resolution: "1920x1080", Bitrate: "4000k", AudioRate: "192k"},
        {Name: "720p",  Resolution: "1280x720",  Bitrate: "2500k", AudioRate: "128k"},
        {Name: "480p",  Resolution: "854x480",   Bitrate: "1000k", AudioRate: "96k"},
    }

    //  semaphore — max 3 qualities running in parallel
    semaphore := make(chan struct{}, 3)

    //  waitgroup — wait for ALL qualities to finish
    var wg sync.WaitGroup

    //  collect errors from goroutines
    errChan := make(chan error, len(qualities))

    for _, q := range qualities {
        wg.Add(1)

        // capture loop variable — critical in Go
        q := q

        go func() {
            defer wg.Done()

            // acquire semaphore slot
            semaphore <- struct{}{}
            defer func() { <-semaphore }() // release when done

            fmt.Printf("started transcoding: %s\n", q.Name)

            err := transcodeQuality(inputFile, videoUploadId, userId, q)
            if err != nil {
                fmt.Printf("failed transcoding %s: %v\n", q.Name, err)
                errChan <- fmt.Errorf("quality %s failed: %w", q.Name, err)
                return
            }

            fmt.Printf("✅ finished transcoding: %s\n", q.Name)
        }()
    }

    // wait for all goroutines to finish
    wg.Wait()
    close(errChan)

    // collect any errors
    var errs []string
    for err := range errChan {
        if err != nil {
            errs = append(errs, err.Error())
        }
    }
    if len(errs) > 0 {
        return fmt.Errorf("transcoding errors: %s", strings.Join(errs, ", "))
    }

    fmt.Println(" all qualities transcoded successfully")
    return nil
}

// separate function for single quality transcoding
func transcodeQuality(inputFile string, videoUploadId int64, userId int64, q Quality) error {
    now := time.Now()
    envs:= env.NewEnv()

    // insert processing record
    details := &models.VideoQuality{
        VideoID:    videoUploadId,
        UserID:     userId,
        Quality:    q.Name,
        Codec:      "H.264",
        Bitrate:    q.Bitrate,
        Resolution: q.Resolution,
        Status:     "processing",
        CreatedAt:  now,
    }

    _, err := connect.Db.NewInsert().
        Model(details).
        Returning("*").
        Exec(context.Background())
    if err != nil {
        return fmt.Errorf("failed to insert quality record: %w", err)
    }

    // create temp dir
    uid := uuid.New()
    qualityDir := fmt.Sprintf("/tmp/%s/%s", uid, q.Name)
    os.MkdirAll(qualityDir, os.ModePerm)
    defer os.RemoveAll(qualityDir) // cleanup after upload

    playlist := fmt.Sprintf("%s/index.m3u8", qualityDir)
    segmentPattern := fmt.Sprintf("%s/segment%%03d.ts", qualityDir)

    // run ffmpeg
    err = ffmpeg.Input(inputFile).
        Output(playlist, ffmpeg.KwArgs{
            "vf":                  fmt.Sprintf("scale=%s", q.Resolution),
            "c:v":                 "libx264",
            "b:v":                 q.Bitrate,
            "c:a":                 "aac",
            "b:a":                 q.AudioRate,
            "preset":              "fast",
            "g":                   "48",
            "keyint_min":          "48",
            "hls_time":            "6",
            "hls_list_size":       "0",
            "hls_segment_filename": segmentPattern,
            "f":                   "hls",
        }).
        OverWriteOutput().
        Run()
    if err != nil {
        // update DB — failed
        connect.Db.NewUpdate().
            Model((*models.VideoQuality)(nil)).
            Set("status = ?", "failed").
            Where("id = ?", details.ID).
            Exec(context.Background())
        return fmt.Errorf("ffmpeg failed: %w", err)
    }

    // upload all files to S3
    files, err := os.ReadDir(qualityDir)
    if err != nil {
        return err
    }

    var playlistS3Key string
    var totalSize int64

    for _, f := range files {
        localFile := filepath.Join(qualityDir, f.Name())

        // get file size
        fileInfo, _ := os.Stat(localFile)
        totalSize += fileInfo.Size()

        s3Key := fmt.Sprintf("transcoded/%s/%s/%s", uid, q.Name, f.Name())

        // track playlist key separately
        if strings.HasSuffix(f.Name(), ".m3u8") {
            playlistS3Key = s3Key
        }

        _, err := utils.UploadToS3(localFile, s3Key)
        if err != nil {
            return fmt.Errorf("upload failed for %s: %w", f.Name(), err)
        }

        fmt.Printf("uploaded: %s\n", s3Key)
    }

    s3Base := envs.S3_BASE_URL
    // "https://your-bucket.s3.ap-south-1.amazonaws.com"

    cdnUrl := fmt.Sprintf("%s/%s", s3Base, playlistS3Key)

    // update DB — completed
    _, err = connect.Db.NewUpdate().
        Model((*models.VideoQuality)(nil)).
        Set("status = ?", "completed").
        Set("playlist_key = ?", playlistS3Key).
        Set("file_size_bytes = ?", totalSize).
        Set("cdn_url = ?", cdnUrl).
        Where("id = ?", details.ID).
        Exec(context.Background())
    if err != nil {
        return fmt.Errorf("failed to update DB: %w", err)
    }

    return nil
}


