package handlers

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"time"

	"viralforge/cmd/worker/tasks"
	"viralforge/src/connect"
	"viralforge/src/env"
	"viralforge/src/models"
	"viralforge/src/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/lib/pq"
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


func GetDownloadUrl() fiber.Handler {
    return func(c fiber.Ctx) error {
		fmt.Println("get download url")
		object_key := c.Query("objectKey")
		fmt.Println("object key: ", object_key)
		// setup AWS config

		presigned_url, err:= utils.DownloadFromS3(object_key)
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

func CreateVideo() fiber.Handler{
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

		fileType:= GetContentType(body.Filename)

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
			FileType: fileType,
			Tags:[]string{},
		}

		_, err = connect.Db.NewInsert().Model(videoUpload).Returning("*").Exec(c.Context())
		fmt.Println("error inserting into db: ", err)
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
			Message: "successfully inserted video file",
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

// userid , objectkey, videoid

func VideoTranscode(userID int64, objectKey string, videoID int64) error {
	fmt.Println("object key: ", objectKey, videoID)
	task, err := tasks.NewTranscodeVideoTask(videoID, objectKey, userID)
	if err != nil {
		return err
	}

	fmt.Println("task: ", task)

	_, err = connect.AsynqClient.Enqueue(task)
	if err != nil {
		return err
	}

	return nil
}

	type Quality struct {
		Name       string
		Resolution string
		Bitrate    string
		AudioRate  string  // ← add this
	}


func GetContentType(filePath string) string {

    ext := strings.ToLower(filepath.Ext(filePath))

    switch ext {

    case ".m3u8":
        return "application/vnd.apple.mpegurl"

    case ".ts":
        return "video/MP2T"

    case ".mp4":
        return "video/mp4"
	case ".mkv":
        return "video/x-matroska"
    default:
        return "application/octet-stream"
    }
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

func GetTranscodedVideoStatus() fiber.Handler{
	return func (c fiber.Ctx) error{
		v_id, _:= strconv.Atoi(c.Params("v_id"))
		

		// get all the videos 
		var v_details []models.VideoQuality 

		err:= connect.Db.NewSelect().Model(&v_details).Where("video_upload_id = ?", v_id).Scan(c.Context())
		if err!=nil{
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"success": false,
				"code":    500,
				
			})
		}

		if len(v_details)==0{
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"success": true,
				"code":    200,
				"status":  "processing",
				"data":    nil,
			})
		}

		allDone:= true 
		for _,q:= range v_details{
			if q.Status !="completed"{
				allDone=false
				break;
			}
		}

		if !allDone{
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
                "success": true,
                "code":    200,
                "status":  "processing",
                "data":    v_details,
            })
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
            "success": true,
            "code":    200,
            "status":  "completed",
            "data":    v_details,
        })
	}
}


func UpdateCDN_Url() fiber.Handler{
	return func (c fiber.Ctx) error{
		envs:= env.NewEnv()

		var v_details []models.VideoQuality 
		err:= connect.Db.NewSelect().Model(&v_details).Where("cdn_url = ?","").Scan(c.Context())
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(err)
		}

		for _,q := range v_details{
		
			playlist_key := q.PlaylistKey 
			s3Base := envs.S3_BASE_URL

			cdnUrl:= fmt.Sprintf("%s/%s", s3Base, playlist_key)

			// now update the db
			_, err:= connect.Db.NewUpdate().Model((*models.VideoQuality)(nil)).Set("cdn_url = ?", cdnUrl).Where("status = ?","completed").Exec(context.Background())

			if err!=nil{
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"message":"failed to update db",
					"success":500,
				})
			}
		}
		return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
			"message":"updated successfully",
			"success":200,
		})
	}
}

type VideoData struct {
	Video  *models.VideoUpload 
	VideoQuality *models.VideoQuality
}
type VideoResponse struct{
	Data VideoData 
	Success bool 
	Code    int
}

type VideoMetaData struct {
	Title       string  `json:"title"`
	Description string	`json:"description"`
	Tags        pq.StringArray `json:"tags"`
	Thumbnail   string	`json:"thumbnail"`
	VideoId     int64	`json:"video_id"`
	ObjectKey   string	`json:"object_key"`
	PublishStatus string `json:"publish_status"`
	
}

type UpdatedVideoData struct{
	Title       string  `json:"title"`
	Description string	`json:"description"`
	Tags        pq.StringArray `json:"tags"`
	Thumbnail   string	`json:"thumbnail"`
	VideoId     int64	`json:"video_id"`
	ObjectKey   string	`json:"object_key"`
	PublishStatus string `json:"publish_status"`
	VideoDuration float64 `json:"video_duration"`
}


func UpdateVideo() fiber.Handler {
	return func(c fiber.Ctx) error {

		userID, err := FetchUserId(c)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(VideoResponse{
				Success: false, Code: 500,
			})
		}

		vID, err := strconv.Atoi(c.Params("v_id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(VideoResponse{
				Success: false, Code: 400,
			})
		}


		var body VideoMetaData
		if err := c.Bind().Body(&body); err != nil {
			fmt.Printf("failed to bind request body: %v\n", err)
			return c.Status(fiber.StatusBadRequest).JSON(VideoResponse{
				Success: false, Code: 400,
			})
		}
		body.VideoId = int64(vID)

		fmt.Println("video id: ", body.VideoId, body)

		// 1st check if the video id exist or not , if not terminate the process 
		video_details,err := connect.Db.NewSelect().Model(&models.VideoUpload{}).Where("id = ?", vID).Exists(c.Context())
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(VideoResponse{
				Success: false, Code: 400,
			})
		}
		if !video_details {
			return c.Status(fiber.StatusBadRequest).JSON(VideoResponse{
				Success: false, Code: 400,
			})
		}
		
		video_duration, _:= utils.GetVideoDuration(body.ObjectKey)
		updatedData := &UpdatedVideoData{
			Title:          body.Title,
			Description:   body.Description,
			Tags:            body.Tags,
			Thumbnail:       body.Thumbnail,
			VideoId:       body.VideoId,
			ObjectKey:     body.ObjectKey,
			PublishStatus: body.PublishStatus,
			VideoDuration: video_duration,
		}
		// Step 1: Update metadata
		if err = UpdateVideoMetadata(c.Context(),updatedData); err != nil {
			fmt.Printf("failed updating metadata: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(VideoResponse{
				Success: false, Code: 500,
			})
		}

		// Step 2: Push async transcode job
		fmt.Println("video id and object key: ", body.VideoId, body.ObjectKey)
		if err = VideoTranscode(userID, body.ObjectKey, body.VideoId); err != nil {
			fmt.Printf("failed to enqueue transcode job: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(VideoResponse{
				Success: false, Code: 500,
			})
		}

		return c.Status(fiber.StatusAccepted).JSON(VideoResponse{
			Success: true, Code: 202,
		})
	}
}

// for updaing published status

func UpdateVideoMetadata(ctx context.Context, data *UpdatedVideoData)error{
	// update the db 
	if data.VideoId == 0 {
		return fmt.Errorf("video id is required")
	}
	res,err := connect.Db.NewUpdate().Model((*models.VideoUpload)(nil)).Where("id = ?", data.VideoId).Set("title = ?", data.Title).Set("description = ?", data.Description).Set("tags = ?", data.Tags).Set("publish_status = ?", data.PublishStatus).Exec(ctx)
	if err!=nil{
		return err
	}
	fmt.Println("result of updated meta data: ", res)
	return nil
}

type DeleteVideoResponse struct{
	Success bool 
	Code    int
	Message string
}


func DeleteVideo() fiber.Handler{
	return func(c fiber.Ctx) error{

		video_id,_:= strconv.Atoi(c.Params("v_id"))
		user_id,_:= FetchUserId(c)

		// all these below operation in queue
		// deleting the date from db and video from the s3  and if transcoded video is present then delete that too
		// 1. get the video data 
		// 2. first delete it from the s3 
		// 3. then get the vide_quality details, get the transcoded key and that too delete it from the s3
		// 4. now delete the data from the db.
		task, err:= tasks.DeleteVideoTask(int64(video_id),user_id)
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(DeleteVideoResponse{
				Success: false,
				Code:400,
				Message:"Failed to queue up delete task",
			})
		}
		_, err = connect.AsynqClient.Enqueue(task)
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(DeleteVideoResponse{
				Success: false,
				Code:400,
				Message:"Failed to queue up delete task",
			})
		}

		// now update the video_upload and vide_quality data - is_deleted : true
		_ , err = connect.Db.NewUpdate().Model((*models.VideoUpload)(nil)).Set("is_deleted = ?", true).Where("id = ?", video_id).Exec(c.Context()) 
		if err!=nil{
			return  c.Status(fiber.StatusInternalServerError).JSON(DeleteVideoResponse{
				Success: false,
				Code:400,
				Message:"Failed to update the video_upload table with delete",
			})
		}

		_ , err = connect.Db.NewUpdate().Model((*models.VideoQuality)(nil)).Set("is_deleted = ?", true).Where("video_upload_id = ?", video_id).Exec(c.Context()) 
		if err!=nil{
			return  c.Status(fiber.StatusInternalServerError).JSON(DeleteVideoResponse{
				Success: false,
				Code:400,
				Message:"Failed to update the video_quality table with delete",
			})
		}

		
		return  c.Status(fiber.StatusAccepted).JSON(DeleteVideoResponse{
			Success: true,
			Code:200,
			Message:"Successfully Deleted the post",
		})
	}
}


type VideoPost struct{
    id 			int64;
    userId 		int64;
    userName	string;
    userAvatar  string;
    userVerified bool;
    title      string;
    description string;
    category  string;
    thumbnail  string;
    duration string;
    views    int64;
    likes   int64;
    quality string;
    tags  []string
    reviews []Review
    time  string;
};

type Review struct{} //comments

type PostedVideoResponse struct{
	Data []VideoPost 
	Success bool 
	Code   int
}

func GetAllPostedVideos() fiber.Handler{
	return func (c fiber.Ctx) error{
		userId, err:= FetchUserId(c);
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"Message":"failed to retrieve user id , log in or signup",
				"Success":false,
			})
		}

		// get the userinfo as well
		var user_info models.User 
		err = connect.Db.NewSelect().Model((&user_info)).Where("id = ?", userId).Scan(c.Context());
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"Message":"Failed to retrieve user info",
				"Success": false,
			})
		}

		// get the the videos of that are published status is "published"
		var posted_videos models.VideoUpload 
		err = connect.Db.NewSelect().Model((&posted_videos)).Where("user_id = ?", userId).Where("published_status = ?","published").Scan(c.Context())
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"Message":"Failed to retrieve videos",
				"Success": false,
			})
		}

		return c.Status(fiber.StatusAccepted).JSON(VideoResponse{

		})




	}
}












