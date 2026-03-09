package handlers

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"viralforge/src/connect"
	"viralforge/src/env"
	"viralforge/src/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)


func GetObjectKey(videoname string) string{

		id := uuid.New().String()
		return fmt.Sprintf("videos/%s-%s", id, videoname)
	
}

type GetPresignedUrlResponse struct {
	Messsage string 
	Url      string
	ObjectKey string
	Code     int16
}

func GetPresignedUrl() fiber.Handler{
	return func(c fiber.Ctx) error{
		envs:= env.NewEnv() 

		aws_access_key:=envs.AWS_ACCESS_KEY_ID
		aws_secret_key:=envs.AWS_SECRET_ACCESS_KEY 
		bucketname := envs.S3_BUCKET_NAME 
		

		cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(aws_access_key,aws_secret_key,"")),
	)
	if err!=nil{
		log.Fatal("Failed to load config")
	}
	var req struct{
		VideoFileKey string `json:"filename"`
		ContentType string  `json:"contentType"`
	}
	if err:=c.Bind().Body(&req); err!=nil{
		return c.Status(fiber.StatusBadRequest).JSON(GetPresignedUrlResponse{
			Messsage: "Failed to fetch the video file. Invalid request body.",
			Code:400,
		})
	}

	client:= s3.NewFromConfig(cfg)
	presignClient := s3.NewPresignClient(client)
	
	
	objectKey:= GetObjectKey(req.VideoFileKey)
	params:=&s3.PutObjectInput{
		Bucket: aws.String(bucketname),
		Key: aws.String(objectKey),
		ContentType:&req.ContentType ,
		// ACL:         types.ObjectCannedACLPublicRead,
	}
	presignedUrl, err:=presignClient.PresignPutObject(context.TODO(), params,func(opts *s3.PresignOptions){
		opts.Expires = time.Hour
	})

	if err!=nil{
		return c.Status(fiber.StatusBadRequest).JSON(GetPresignedUrlResponse{
			Messsage: "Failed to generate presignedUrl",
			Code:500,

		})
	}

	return c.Status(fiber.StatusOK).JSON(GetPresignedUrlResponse{
		Messsage: "Successful with presigned url",
		Url: presignedUrl.URL,
		ObjectKey:objectKey,
		Code: 200,
	})

	}
}


func DownloadFromS3( objectKey string) (string, error){
	envs:=env.NewEnv()
	fmt.Println("envs:")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
            config.WithRegion("us-east-1"),
            config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
                envs.AWS_ACCESS_KEY_ID,
                envs.AWS_SECRET_ACCESS_KEY,
                "",
            )),
        )
        if err != nil {
            return "failed to load aws config", err
        }

		presignClient := s3.NewPresignClient(s3.NewFromConfig(cfg))
        presignedUrl, err := presignClient.PresignGetObject(context.TODO(),
            &s3.GetObjectInput{
                Bucket: aws.String(envs.S3_BUCKET_NAME),
                Key:    aws.String(objectKey),
				
            },
            func(opts *s3.PresignOptions) {
                opts.Expires = 15 * time.Minute
            },
        )
        if err != nil {
            return "failed to generate download url", err
        }

		return presignedUrl.URL, err

}

func GetDownloadUrl() fiber.Handler {
    return func(c fiber.Ctx) error {
		fmt.Println("get download url")
		object_key := c.Query("objectKey")
		fmt.Println("object key: ", object_key)
		// setup AWS config

		presigned_url, err:= DownloadFromS3(object_key)
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"Message":"Failed to generate presigned url",
			})
		}
        
        return c.Status(fiber.StatusOK).JSON(fiber.Map{
            "url": presigned_url,
        })
    }
}

// get the video and then start transcoding
type VideoUploadResponse struct{
	Data 	models.VideoUpload
	Code    int64
	Success  bool
	Message  string
}

//TODO: add uploaded video to job queue ( not directly to the db , worker will do the job of inserting the video to s3 , optional : in transcoding job, start transcoding) for uploading videos to the s3 and do not wan
func AddVideoFileKeyToDB() fiber.Handler{
	return func(c fiber.Ctx) error{
		u_id, err:=FetchUserId(c)
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(VideoUploadResponse{
				Data:models.VideoUpload{},
				Code:400,
				Success:false,
				Message: "Failed to fetch userid",
			})
		}
		// ❌ Wrong — lowercase fields, JSON can't see them
		// var body struct {
		// 	filename  string
		// 	fileType  string
		// 	objectKey string
		// }

		var body struct {
			Filename  string `json:"filename"`
			FileType  string `json:"fileType"`
			ObjectKey string `json:"objectKey"`
		}

		if err:=c.Bind().Body(&body); err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(VideoUploadResponse{
				Data:models.VideoUpload{},
				Code:400,
				Success:false,
				Message: "Failed request body",
			})
		}

		if(body.ObjectKey==""){
			return c.Status(fiber.StatusBadRequest).JSON(VideoUploadResponse{
				Data:models.VideoUpload{},
				Code:400,
				Success:false,
				Message: "body is empty",
			})
		}

		fmt.Println("body for create video: ", body)

		
		videoUpload:=&models.VideoUpload{
			UserID: u_id,
			FileURL: body.ObjectKey,
			FileType: body.FileType,
		}

		_, err = connect.Db.NewInsert().Model(videoUpload).Returning("*").Exec(c.Context())
		
		if err!=nil{
			fmt.Println("error while inserting it into db")
			return c.Status(fiber.StatusBadRequest).JSON(VideoUploadResponse{
				Data:models.VideoUpload{},
				Code:400,
				Success:false,
				Message: "Failed to insert vidoe file",
			})
		}

		return c.Status(fiber.StatusAccepted).JSON(VideoUploadResponse{
			Data:*videoUpload,
			Code:200,
			Success:true,
			Message: "successfully inserted vidoe file",
		})

	}
}

type GetListOfVideoFilesResponse struct{
	VideoFiles	[]models.VideoUpload 
	Success 	bool 
	Code        int32
}
func GetListOfVideoFiles() fiber.Handler{
	return func (c fiber.Ctx) error{
	
		u_id, err:= FetchUserId(c)
		
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(GetListOfVideoFilesResponse{
				VideoFiles: []models.VideoUpload{},
				Success: false,
				Code:400,
			})
		}
	
		var videoFiles []models.VideoUpload
		
		err = connect.Db.NewSelect().
			Model(&videoFiles).
			Where("user_id = ?", u_id).
			Scan(c.Context())

		
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(GetListOfVideoFilesResponse{
				VideoFiles: []models.VideoUpload{},
				Success: false,
				Code:400,
			})
		}

		return c.Status(fiber.StatusAccepted).JSON(GetListOfVideoFilesResponse{
			VideoFiles: videoFiles,
			Success: true,
			Code:200,
		})
	}
}

type VideoTranscodeResponse struct{
	Data  *[]models.VideoQuality 
	Success bool 
	Code   int
}

func VideoTranscode() fiber.Handler{
	return func(c fiber.Ctx) error{
		user_id, err := FetchUserId(c) 
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON("failed to fetch user id")
		}
		object_key := c.Query("objectKey")
		videoId,_ := strconv.Atoi(c.Query("videoId"))
		v_id := int64(videoId)

		err = HLSTranscode(v_id,object_key, user_id)
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}
		
		var v_details []models.VideoQuality
		err=connect.Db.NewSelect().Model((&v_details)).Where("video_upload_id = ?",videoId).Scan(c.Context())

		if err!=nil{
			return c.Status(fiber.StatusInternalServerError).JSON(VideoTranscodeResponse{
				Success: false,
				Code:500,
			})
		}

		return c.Status(fiber.StatusOK).JSON(VideoTranscodeResponse{
			Data:&v_details,
			Success: true,
			Code:200,
		})
	}
}

	type Quality struct {
		Name       string
		Resolution string
		Bitrate    string
		AudioRate  string  // ← add this
	}


func getContentType(filePath string) string {

    ext := strings.ToLower(filepath.Ext(filePath))

    switch ext {

    case ".m3u8":
        return "application/vnd.apple.mpegurl"

    case ".ts":
        return "video/MP2T"

    case ".mp4":
        return "video/mp4"

    default:
        return "application/octet-stream"
    }
}
	
func UploadToS3(localFilePath string, s3Key string)(bool, error) {
    envs := env.NewEnv()

    cfg, err := config.LoadDefaultConfig(context.TODO(),
        config.WithRegion("us-east-1"),
        config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
            envs.AWS_ACCESS_KEY_ID,
            envs.AWS_SECRET_ACCESS_KEY,
            "",
        )),
    )
    if err != nil {
        return false,fmt.Errorf("failed to load aws config: %w", err)
    }

    // open the local file
    file, err := os.Open(localFilePath)
    if err != nil {
        return false,fmt.Errorf("failed to open local file: %w", err)
    }
    defer file.Close()

    // get file size
    fileInfo, err := file.Stat()
    if err != nil {
        return false,fmt.Errorf("failed to get file info: %w", err)
    }

    s3Client := s3.NewFromConfig(cfg)

    // use multipart uploader for large files
    uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {
        u.PartSize = 10 * 1024 * 1024 // 10MB per part
        u.Concurrency = 3              // 3 parallel uploads
    })

	contentType := getContentType(localFilePath)

    _, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
        Bucket:        aws.String(envs.S3_BUCKET_NAME),
        Key:           aws.String(s3Key),
        Body:          file,
        ContentType:   aws.String(contentType),
        ContentLength: aws.Int64(fileInfo.Size()),
    })
    if err != nil {
        return false,fmt.Errorf("failed to upload to S3: %w", err)
    }

    fmt.Printf("successfully uploaded %s to S3 at %s\n", localFilePath, s3Key)
    return true,nil
}

type GetTheVideoDetailsUploadedResponse struct {
	Data  *models.VideoQuality 
	Success bool 
	Code    int 
}

func GetTranscodedVideoDetails() fiber.Handler{
	return func (c fiber.Ctx) error{
		videoId:= c.Query("videoId")

		var video_transcoded_details models.VideoQuality 
		err:= connect.Db.NewSelect().Model(&video_transcoded_details).Where("video_upload_id = ?", videoId).Scan(c.Context())
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(GetTheVideoDetailsUploadedResponse{
				Success: false,
				Code:400,
			})
		}

		return c.Status(fiber.StatusOK).JSON(GetTheVideoDetailsUploadedResponse{
			Data:&video_transcoded_details,
			Success: true,
			Code:200,
		})
	}
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
    inputFile, err := DownloadFromS3(inputKey)
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

        _, err := UploadToS3(localFile, s3Key)
        if err != nil {
            return fmt.Errorf("upload failed for %s: %w", f.Name(), err)
        }

        fmt.Printf("uploaded: %s\n", s3Key)
    }

    // update DB — completed
    _, err = connect.Db.NewUpdate().
        Model((*models.VideoQuality)(nil)).
        Set("status = ?", "completed").
        Set("playlist_key = ?", playlistS3Key).
        Set("file_size_bytes = ?", totalSize).
        Where("id = ?", details.ID).
        Exec(context.Background())
    if err != nil {
        return fmt.Errorf("failed to update DB: %w", err)
    }

    return nil
}






