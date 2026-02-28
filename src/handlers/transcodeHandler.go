package handlers

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"
	"viralforge/src/connect"
	"viralforge/src/env"
	"viralforge/src/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
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
		envs:= env.NewEnv()
		object_key := c.Query("objectKey")
		fmt.Println("object key: ", object_key)
		// setup AWS config

		
        cfg, err := config.LoadDefaultConfig(context.TODO(),
            config.WithRegion("us-east-1"),
            config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
                envs.AWS_ACCESS_KEY_ID,
                envs.AWS_SECRET_ACCESS_KEY,
                "",
            )),
        )
        if err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "message": "failed to load aws config",
            })
        }

        presignClient := s3.NewPresignClient(s3.NewFromConfig(cfg))
        presignedUrl, err := presignClient.PresignGetObject(context.TODO(),
            &s3.GetObjectInput{
                Bucket: aws.String(envs.S3_BUCKET_NAME),
                Key:    aws.String(object_key),
				
            },
            func(opts *s3.PresignOptions) {
                opts.Expires = 15 * time.Minute
            },
        )
        if err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "message": "failed to generate download url",
            })
        }

        return c.Status(fiber.StatusOK).JSON(fiber.Map{
            "url": presignedUrl.URL,
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

		userid,_:= strconv.Atoi(u_id)
		videoUpload:=&models.VideoUpload{
			UserID: int64(userid),
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
		userID, err := strconv.ParseInt(u_id, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "invalid user id",
			})
		}

		
		var videoFiles []models.VideoUpload
		
		err = connect.Db.NewSelect().
			Model(&videoFiles).
			Where("user_id = ?", userID).
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

