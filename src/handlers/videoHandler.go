package handlers

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
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
		region:=envs.S3_REGION

		cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
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

		_, err = connect.Db.NewInsert().Model(videoUpload).Exec(c.Context())
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
			fmt.Printf("all done - uploading")
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
                "success": true,
                "code":    200,
                "status":  "uploading", // processing
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
	Title       string  `form:"title"`
	Description string	`form:"description"`
	Tags        string `form:"tags"`
	Thumbnail   string	`form:"thumbnail"`
	VideoId     int64	`form:"video_id"`
	ObjectKey   string	`form:"object_key"`
	PublishStatus string `form:"publish_status"`
	
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
		if err := c.Bind().Form(&body); err != nil {
			fmt.Printf("failed to bind request body: %v\n", err)
			return c.Status(fiber.StatusBadRequest).JSON(VideoResponse{
				Success: false, Code: 400,
			})
		}
		body.VideoId = int64(vID)

		fmt.Println("video id: ", body.VideoId, body)

		// 1st check if the video id exist or not , if not terminate the process 
		var video models.VideoUpload

		err = connect.Db.NewSelect().
		Model(&video).
		Where("id = ?", vID).
		Limit(1).
		Scan(c.Context())

		if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(VideoResponse{
				Success: false,
				Code:    404,
				
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(VideoResponse{
			Success: false,
			Code:    500,
		
		})
		}
		// get the video details and change the data which need to be changed
		if body.Title == "" {
			body.Title = video.Title
		}

		if body.Description == "" {
			body.Description = video.Description
		}

		if body.Thumbnail == "" {
			body.Thumbnail = video.SelectedThumbnail
		}

		if body.ObjectKey == "" {
			body.ObjectKey = video.FileURL
		}
		
		if body.PublishStatus == ""{
			body.PublishStatus = "draft"
		}

		var tags []string
		if err := json.Unmarshal([]byte(body.Tags), &tags); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "invalid tags",
			})
		}
		
		
		video_duration, _:= utils.GetVideoDuration(body.ObjectKey)
		updatedData := &UpdatedVideoData{
			Title:          body.Title,
			Description:   body.Description,
			Tags:            pq.StringArray(tags),
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


	res,err := connect.Db.NewUpdate().Model((*models.VideoUpload)(nil)).Where("id = ?", data.VideoId).Set("title = ?", data.Title).Set("description = ?", data.Description).Set("tags = ?", data.Tags).Set("publish_status = ?", data.PublishStatus).Set("video_duration = ?", data.VideoDuration).Exec(ctx)
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
		v_id := int64(video_id)

		// first check here, if it exists or not 
		exists, err := connect.Db.NewSelect().
        Model((*models.VideoUpload)(nil)).
        Where("id = ?", v_id).
        Exists(c.Context())
		if err != nil {
			return fmt.Errorf("checking video existence: %w", err)
		}

		if !exists{
			return  c.Status(fiber.StatusAccepted).JSON(DeleteVideoResponse{
				Success: true,
				Code:200,
				Message:"Video id does not exist",
			})
		}
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


type VideoPost struct {
    ID          int64           `json:"id"`
    UserID      int64           `json:"userId"`
    UserName    string          `json:"userName"`
    Title       string          `json:"title"`
    Description string          `json:"description"`
    Tags        []string        `json:"tags"`
    Views       int64           `json:"views"`
    Likes       int64           `json:"likes"`
    Thumbnail   string          `json:"thumbnail"`
	Duration    float64         `json:"duration"`
	Category    string          `json:"category"`
    Time        string          `json:"time"`
    Qualities   []QualityOption `json:"qualities"`
}
type QualityOption struct {
    CDNUrl  string `json:"cdnUrl"`
    Quality string	`json:"quality"`
}

type Review struct{} //comments

type PostedVideoResponse struct{
	Message string
	Data []VideoPost 
	Success bool 
	Code   int
}


type VideoFeedRow struct {
    ID          int64          `bun:"id"`
    Title       string         `bun:"title"`
    Description string         `bun:"description"`
    Tags        pq.StringArray `bun:"tags"`
    ViewsCount  int64          `bun:"views_count"`
    LikesCount  int64          `bun:"likes_count"`
    Thumbnail   string         `bun:"selected_thumbnail"`
    CreatedAt   time.Time      `bun:"created_at"`
	Duration    float64        `bun:"video_duration"`
	Category    string         `bun:"category"`
	

    UserID   int64  `bun:"user_id"`
    UserName string `bun:"user_name"`

    CDNUrl  string `bun:"cdn_url"`
    Quality string `bun:"quality"`
}


func GetAllPostedVideosOfUser() fiber.Handler {
	return func(c fiber.Ctx) error {

		// ✅ Fetch user ID
		userID, err := FetchUserId(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(PostedVideoResponse{
				Message: "Login required to view your videos",
				Success: false,
				Code:    401,
			})
		}

		// ✅ Query params
		cursor := c.Query("cursor")
		limit := fiber.Query[int](c, "limit", 10)

		var createdAtCursor time.Time
		var idCursor int64

		// ✅ Decode cursor safely
		if cursor != "" {
			decoded, err := base64.StdEncoding.DecodeString(cursor)
			if err == nil {
				parts := strings.Split(string(decoded), "|")
				if len(parts) == 2 {
					createdAtCursor, _ = time.Parse(time.RFC3339, parts[0])
					idCursor, _ = strconv.ParseInt(parts[1], 10, 64)
				}
			}
		}

		var rows []VideoFeedRow

		// ✅ Clean query
		query := connect.Db.NewSelect().
			TableExpr("video_uploads AS vdu").
			Column(
				"vdu.id",
				"vdu.title",
				"vdu.description",
				"vdu.tags",
				"vdu.views_count",
				"vdu.likes_count",
				"vdu.selected_thumbnail",
				"vdu.created_at",
				"vdu.video_duration",
				"vdu.category",
			).
			ColumnExpr("u.id AS user_id, u.name AS user_name").
			ColumnExpr("vq.cdn_url, vq.quality").
			Join("JOIN users u ON u.id = vdu.user_id").
			Join("LEFT JOIN video_qualities vq ON vq.video_upload_id = vdu.id AND vq.status = ? AND vq.is_deleted = false", "completed").
			Where("vdu.is_deleted = false").
			Where("u.id = ?", userID)

		// ✅ Cursor pagination condition
		if cursor != "" {
			query = query.Where(`
				(vdu.created_at < ?) OR
				(vdu.created_at = ? AND vdu.id < ?)
			`, createdAtCursor, createdAtCursor, idCursor)
		}

		// ✅ Execute query
		err = query.
			OrderExpr("vdu.created_at DESC, vdu.id DESC").
			Limit(limit + 1).
			Scan(c.Context(), &rows)

		if err != nil {
			fmt.Println("DB error:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(PostedVideoResponse{
				Message: "Failed to fetch video details",
				Success: false,
				Code:    500,
			})
		}

		// ✅ Group results
		videoMap := make(map[int64]*VideoPost)
		videoOrder := []int64{}

		for _, r := range rows {
			if _, exists := videoMap[r.ID]; !exists {
				videoOrder = append(videoOrder, r.ID)

				videoMap[r.ID] = &VideoPost{
					ID:          r.ID,
					UserID:      r.UserID,
					UserName:    r.UserName,
					Title:       r.Title,
					Description: r.Description,
					Tags:        r.Tags,
					Views:       r.ViewsCount,
					Likes:       r.LikesCount,
					Thumbnail:   r.Thumbnail,
					Duration:    r.Duration,
					Category:    r.Category,
					Time:        r.CreatedAt.Format(time.RFC3339),
					Qualities:   []QualityOption{},
				}
			}

			// ✅ Append qualities
			if r.CDNUrl != "" {
				videoMap[r.ID].Qualities = append(videoMap[r.ID].Qualities, QualityOption{
					CDNUrl:  r.CDNUrl,
					Quality: r.Quality,
				})
			}
		}

		// ✅ Prepare response list
		result := []VideoPost{}
		for i, id := range videoOrder {
			if i >= limit {
				break
			}
			result = append(result, *videoMap[id])
		}

		// ✅ Pagination metadata
		hasMore := len(videoOrder) > limit
		var nextCursor string

		if hasMore {
			last := videoMap[videoOrder[limit-1]]
			cursorStr := fmt.Sprintf("%s|%d", last.Time, last.ID)
			nextCursor = base64.StdEncoding.EncodeToString([]byte(cursorStr))
		}

		// ✅ Final response
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"Data":       result,
			"NextCursor": nextCursor,
			"HasMore":    hasMore,
			"Success":    true,
			"Code":       200,
		})
	}
}


func GetAllPostVideosOfPlatform() fiber.Handler {
    return func(c fiber.Ctx) error {

		// get query params:
		cursor := c.Query("cursor") // base64 encoded
        limit := fiber.Query[int](c,"limit", 10)

		var createdAtCursor time.Time
        var idCursor int64

		// 2. Decode cursor (if exists)
        if cursor != "" {
            decoded, err := base64.StdEncoding.DecodeString(cursor)
            if err == nil {
                parts := strings.Split(string(decoded), "|")
                if len(parts) == 2 {
                    createdAtCursor, _ = time.Parse(time.RFC3339, parts[0])
                    idCursor, _ = strconv.ParseInt(parts[1], 10, 64)
                }
            }
        }

        var rows []VideoFeedRow

        query := connect.Db.NewSelect().
			TableExpr("video_uploads AS vdu").
			Column(
				"vdu.id",
				"vdu.title",
				"vdu.description",
				"vdu.tags",
				"vdu.views_count",
				"vdu.likes_count",
				"vdu.selected_thumbnail",
				"vdu.created_at",
				"vdu.video_duration",
				"vdu.category",
			).
			ColumnExpr("u.id AS user_id, u.name AS user_name").
			ColumnExpr("vq.cdn_url, vq.quality").
			Join("JOIN users u ON u.id = vdu.user_id").
			Join("LEFT JOIN video_qualities vq ON vq.video_upload_id = vdu.id AND vq.status = ? AND vq.is_deleted = false", "completed").
			Where("vdu.publish_status = ?", "published").
			Where("vdu.is_deleted = false")
			

		// cursor condition
		if cursor != "" {
            query = query.Where(`
                (vdu.created_at < ?) OR 
                (vdu.created_at = ? AND vdu.id < ?)
            `, createdAtCursor, createdAtCursor, idCursor)
        }
		err := query.
            OrderExpr("vdu.created_at DESC, vdu.id DESC").
            Limit(limit + 1). // +1 to check hasMore
            Scan(c.Context(), &rows)
		fmt.Println("rows: ", rows)

        if err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(PostedVideoResponse{
                Message: "Failed to fetch video details",
                Success: false,
                Code:    500,
            })
        }

        // Debug: inspect raw rows
        // for _, row := range rows {
        //     b, _ := json.MarshalIndent(row, "", "  ")
        //     fmt.Println(string(b))
        // }

        // Group by video ID, preserving order
        videoMap := make(map[int64]*VideoPost)
        videoOrder := []int64{}

        for _, r := range rows {
            if _, exists := videoMap[r.ID]; !exists {
                videoOrder = append(videoOrder, r.ID)
                videoMap[r.ID] = &VideoPost{
                    ID:          r.ID,
                    UserID:      r.UserID,
                    UserName:    r.UserName,
                    Title:       r.Title,
                    Description: r.Description,
                    Tags:        r.Tags,
                    Views:       r.ViewsCount,
                    Likes:       r.LikesCount,
                    Thumbnail:   r.Thumbnail,
					Category:    r.Category,
                    Time:        r.CreatedAt.Format(time.RFC3339),
                    Qualities:   []QualityOption{},
                }
            }
            if r.CDNUrl != "" {
                videoMap[r.ID].Qualities = append(videoMap[r.ID].Qualities, QualityOption{
                    CDNUrl:  r.CDNUrl,
                    Quality: r.Quality,
                })
            }
        }

        result := []VideoPost{}
        for i, id := range videoOrder {
			if i>=limit{
				break
			}
            result = append(result, *videoMap[id])
        }

		hasMore:=len(videoOrder)>limit
		var nextCursor string
        if hasMore {
            last := videoMap[videoOrder[limit-1]]
            cursorStr := fmt.Sprintf("%s|%d", last.Time, last.ID)
            nextCursor = base64.StdEncoding.EncodeToString([]byte(cursorStr))
        }

        return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
            "data":       result,
            "nextCursor": nextCursor,
            "hasMore":    hasMore,
			"Success":    true,
			"Code":       200,
        })
    }
}

// ?q=ai reels&page=1&limit=20&sort=trending 

type SearchVideosResponse struct{
	Data	*[]models.VideoUpload 
	Success  bool 
	Code     int
	

}

func SearchVideos() fiber.Handler{
	return func(c fiber.Ctx) error{
		queries := c.Queries()
		fmt.Print("queries : ", queries, queries["q"])
		return c.Status(fiber.StatusBadRequest).JSON(SearchVideosResponse{
			Success: true,
			Code : 200,
		})
	}
}













