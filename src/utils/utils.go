package utils

// downloads3
// uploadtos3

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"viralforge/src/env"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)


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

func getContentType(filePath string) string {
    ext := strings.ToLower(filepath.Ext(filePath))

    switch ext {
    case ".m3u8":
        return "application/vnd.apple.mpegurl"
    case ".ts":
        return "video/MP2T"
    case ".mp4":
        return "video/mp4"
    case ".jpg", ".jpeg":
        return "image/jpeg"   // ← add this for thumbnails
    case ".png":
        return "image/png"
    case ".webp":
        return "image/webp"
    default:
        return "application/octet-stream"
    }
}