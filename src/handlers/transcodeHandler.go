package handlers

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	"github.com/uptrace/bun/dialect/pgdialect"
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
	Data  *models.VideoDetailsUpload 
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


		err = TranscodeVideo(v_id, object_key, user_id)
		if err!=nil{
			connect.Db.NewUpdate().
			Model((*models.VideoDetailsUpload)(nil)).
			Set("status = ?", "failed").
			Set("processing_error = ?", err.Error()).
			Where("id = ?", v_id).
			Exec(context.Background())
	
			return c.Status(fiber.StatusInternalServerError).JSON(VideoTranscodeResponse{
				Success: false,
				Code:500,
			})
		}
		var v_details models.VideoDetailsUpload
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
	Resolution  string 
	Bitrate 	string 
	Name 		string
}

func TranscodeVideo(videoUploadID int64, inputKey string, userId int64) error {
    inputFile, err := DownloadFromS3(inputKey)
    if err != nil {
        return err
    }
	fmt.Println("input file: ", inputFile, videoUploadID)
    defer os.Remove(inputFile)

    // create VideoDetailsUpload record with "processing" status
    now := time.Now()
    details := &models.VideoDetailsUpload{

        VideoUploadID: videoUploadID, // ← FK to VideoUpload
		UserID: userId,
        Status:        "processing",
        UploadedAt:    now,
		TranscodedUrls: []string{},
    }

	fmt.Println("details: ", details)

    result, err := connect.Db.NewInsert().
        Model(details).
        Returning("*").
        Exec(context.Background())

	fmt.Println("result: ", result)
	fmt.Println("error :", err)
    if err != nil {
        return fmt.Errorf("failed to create details record: %w", err)
    }

    qualities := []Quality{
        {Resolution: "1920x1080", Bitrate: "4000k", Name: "1080p"},
        {Resolution: "1280x720",  Bitrate: "2500k", Name: "720p"},
        {Resolution: "854x480",   Bitrate: "1000k", Name: "480p"},
    }

    uid := uuid.New().String() // ← move outside loop, same prefix for all qualities
    var transcodedUrls []string

    for _, q := range qualities {
        localOutput := fmt.Sprintf("/tmp/%s-%s.mp4", uid, q.Name)
        outputKey := fmt.Sprintf("transcoded/%s-%s.mp4", uid, q.Name)
		fmt.Println("localoutput , outputkey: ", localOutput, outputKey)
        defer os.Remove(localOutput)

        err := ffmpeg.Input(inputFile).
            Output(localOutput, ffmpeg.KwArgs{
                "vf":     fmt.Sprintf("scale=%s", q.Resolution),
                "b:v":    q.Bitrate,
                "c:v":    "libx264",
                "c:a":    "aac",
                "preset": "fast",
                "crf":    "23",
            }).
            OverWriteOutput().
            Run()
        if err != nil {
            // ← update status to failed with error
            connect.Db.NewUpdate().
                Model((*models.VideoDetailsUpload)(nil)).
                Set("status = ?", "failed").
                Set("processing_error = ?", err.Error()).
                Where("id = ?", details.ID).
                Exec(context.Background())
            return fmt.Errorf("transcoding failed for %s: %w", q.Name, err)
        }

		fmt.Println("local output and output key: ", localOutput, outputKey)

        // ← check upload error
        if err := UploadToS3(localOutput, outputKey); err != nil {
            return fmt.Errorf("upload failed for %s: %w", q.Name, err)
        }

        transcodedUrls = append(transcodedUrls, outputKey)
    }

    // update DB with completed status and transcoded URLs
    processedAt := time.Now()
    _, err = connect.Db.NewUpdate().
        Model((*models.VideoDetailsUpload)(nil)).
        Set("transcoded_urls = ?", pgdialect.Array(transcodedUrls)).
        Set("status = ?", "completed").
        Set("processed_at = ?", processedAt).
        Where("id = ?", details.ID). // ← use details.ID not videoUploadID
        Exec(context.Background())
    if err != nil {
        return fmt.Errorf("failed to update DB: %w", err)
    }

    return nil
}

func UploadToS3(localFilePath string, s3Key string) error {
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
        return fmt.Errorf("failed to load aws config: %w", err)
    }

    // open the local file
    file, err := os.Open(localFilePath)
    if err != nil {
        return fmt.Errorf("failed to open local file: %w", err)
    }
    defer file.Close()

    // get file size
    fileInfo, err := file.Stat()
    if err != nil {
        return fmt.Errorf("failed to get file info: %w", err)
    }

    s3Client := s3.NewFromConfig(cfg)

    // use multipart uploader for large files
    uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {
        u.PartSize = 10 * 1024 * 1024 // 10MB per part
        u.Concurrency = 3              // 3 parallel uploads
    })

    _, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
        Bucket:        aws.String(envs.S3_BUCKET_NAME),
        Key:           aws.String(s3Key),
        Body:          file,
        ContentType:   aws.String("video/mp4"),
        ContentLength: aws.Int64(fileInfo.Size()),
    })
    if err != nil {
        return fmt.Errorf("failed to upload to S3: %w", err)
    }

    fmt.Printf("successfully uploaded %s to S3 at %s\n", localFilePath, s3Key)
    return nil
}

type GetTheVideoDetailsUploadedResponse struct {
	Data  *models.VideoDetailsUpload 
	Success bool 
	Code    int 
}

func GetTranscodedVideoDetails() fiber.Handler{
	return func (c fiber.Ctx) error{
		videoId:= c.Query("videoId")

		var video_transcoded_details models.VideoDetailsUpload 
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


// extract the frames and use ai to search the locations 



// 1. Get video object key from database
// 2. Generate S3 presigned URL (temporary download link)
// 3. Stream video from S3 to temporary file
// 4. Detect scenes using FFmpeg
// 5. Extract keyframes at scene timestamps
// 6. Score frames using OCR
// 7. Analyze top frames with AI
// 8. Clean up temporary files
// 9. Save results to database
// get video duration 


// get the extract frames 
// func(ke *KeyframeExtractor) Extract() ([]float64, error){
// 	cmd := exec.Command("ffmpeg",)
// }

// get the transcoded video:
func ExtractScenes(objectKey string)(error){
	videoPath, err:= DownloadFromS3(objectKey)
	if err!=nil{
		return fmt.Errorf("failed to download: ", err)
	}

	// if this object key has already been analyzed then do not do again
	// if want to update and start analysing or insert the data manually 
	
	// get video duration 
	duration, err := GetVideoDuration(videoPath) 
	if err!=nil{
		return fmt.Errorf("failed to get the video duration: %w", err)
	}
	fmt.Println("duration of the video: ", duration)
	// create output folder for the scence
	outputDir := fmt.Sprintf("/tmp/scenes_%d", time.Now().Unix())

	err = os.MkdirAll(outputDir, os.ModePerm)
	if err!=nil{
		return err
	}

	// ffmpeg scene detection:
	cmd := exec.Command(
		"ffmpeg",
		"-i", videoPath,
		"-filter:v", "select=gt(scene\\,0.4)",
		"-vsync", "vfr",
		filepath.Join(outputDir, "scene_%03d.jpg"),
	)
	
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("scene extraction failed: %w", err)
	}

	fmt.Println("Scenes extracted to:", outputDir)

	return nil

}

func GetVideoDuration(videoPath string)(float64, error){
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	durationStr := strings.TrimSpace(string(output))

	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, err
	}

	return duration, nil
}

