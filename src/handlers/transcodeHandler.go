package handlers

import (
	"context"
	"fmt"
	"log"
	"time"
	"viralforge/src/env"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)


func GetObjectKey(videoname string) string{
	
		id := uuid.New().String()
		return fmt.Sprintf("uploads/%s-%s", id, videoname)
	
}

type GetPresignedUrlResponse struct {
	Messsage string 
	Url      string
}

func GetPresignedUrl() fiber.Handler{
	return func(c fiber.Ctx) error{
		envs:= env.NewEnv() 

		aws_access_key:=envs.AWS_ACCESS_KEY_ID
		aws_secret_key:=envs.AWS_SECRET_ACCESS_KEY 
		bucketname := envs.S3_BUCKET_NAME 

		cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(aws_access_key, aws_secret_key, "")),
	)
	if err!=nil{
		log.Fatal("Failed to load config")
	}
	var req struct{
		VideoFileKey string `json:"videoFileKey"`
	}
	if err:=c.Bind().Body(&req); err!=nil{
		return c.Status(fiber.StatusBadRequest).JSON(GetPresignedUrlResponse{
			Messsage: "Failed to fetch the video file. Invalid request body.",
		})
	}

	client:= s3.NewFromConfig(cfg)
	presignClient := s3.NewPresignClient(client)
	
	objectKey:= GetObjectKey(req.VideoFileKey)
	params:=&s3.PutObjectInput{
		Bucket: aws.String(bucketname),
		Key: aws.String(objectKey),
	}
	presignedUrl, err:=presignClient.PresignPutObject(context.TODO(), params,func(opts *s3.PresignOptions){
		opts.Expires = time.Hour
	})

	if err!=nil{
		return c.Status(fiber.StatusBadRequest).JSON(GetPresignedUrlResponse{
			Messsage: "Failed to generate presignedUrl",

		})
	}

	return c.Status(fiber.StatusOK).JSON(GetPresignedUrlResponse{
		Messsage: "Successful with presigned url",
		Url: presignedUrl.URL,
	})

	}
}

// get the video and then start transcoding