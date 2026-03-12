package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"viralforge/src/utils"

	"github.com/google/uuid"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)
type ThumbnailOptions struct {
    AtSecond   float64  // which second to extract frame from
    Width      int      // resize width  (e.g. 1280)
    Height     int      // resize height (e.g. 720)
}

func ExtractThumbnail(inputFile string, opts ThumbnailOptions)(string, error){
	
	// directory for thumbnails
	thumbDir:=fmt.Sprintf("/tmp/thumb-%s", uuid.New().String())
	os.MkdirAll(thumbDir,os.ModePerm)
	thumbFile:= filepath.Join(thumbDir, "thumbnail.jpg")


	atSecond:= opts.AtSecond
	if atSecond==0{
		atSecond=3.0
	}
	// scale filter
	scaleFiler:=""
	if opts.Height>0 && opts.Width>0{
		scaleFiler = fmt.Sprintf("scale=%d:%d", opts.Width,opts.Height)
	}else{
		scaleFiler="scale:=1280:720"
	}

	err:= ffmpeg.Input(inputFile,ffmpeg.KwArgs{
		"ss" : atSecond,
	}).Output(thumbFile,ffmpeg.KwArgs{
		"vframes":1,
		"q:v":2,
		"vf":scaleFiler,
	}).OverWriteOutput().Run()

	if err != nil {
        os.RemoveAll(thumbDir)
        return "", fmt.Errorf("thumbnail extraction failed: %w", err)
    }

    return thumbFile, nil

}

func ExtractMultipleThumbnail(inputFile string, count int, videoDuration float64)([]string, error){
	thumbDir := fmt.Sprintf("/tmp/thumbs-%s", uuid.New().String())
    os.MkdirAll(thumbDir, os.ModePerm)

	interval := videoDuration / float64(count+1)

	var thumbFiles []string 

	for i := 1; i <= count; i++ {
        atSecond := interval * float64(i)
        thumbFile := filepath.Join(thumbDir, fmt.Sprintf("thumb_%d.jpg", i))

        err := ffmpeg.Input(inputFile, ffmpeg.KwArgs{
            "ss": atSecond,
        }).
        Output(thumbFile, ffmpeg.KwArgs{
            "vframes": 1,
            "q:v":     2,
            "vf":      "scale=1280:720",
        }).
        OverWriteOutput().
        Run()

        if err != nil {
            continue  // skip failed frames, don't stop
        }

        thumbFiles = append(thumbFiles, thumbFile)
    }
	if len(thumbFiles) == 0 {
        return nil, fmt.Errorf("no thumbnails extracted")
    }

    return thumbFiles, nil
}


func GetVideoDuration(inputFile string)(float64, error){
	out, err := exec.Command(
        "ffprobe",
        "-v", "quiet",
        "-print_format", "json",
        "-show_format",
        inputFile,
    ).Output()

	if err != nil {
        return 0, fmt.Errorf("ffprobe failed: %w", err)
    }

    var result struct {
        Format struct {
            Duration string `json:"duration"`
        } `json:"format"`
    }

	if err := json.Unmarshal(out, &result); err != nil {
        return 0, err
    }

    duration, err := strconv.ParseFloat(
        strings.TrimSpace(result.Format.Duration), 64,
    )
    return duration, err
}

func UploadThumbnails(thumbFiles []string, videoUploadId int64) ([]string, error) {

    var cdnUrls []string
    s3Base := os.Getenv("S3_BASE_URL")

    for i, thumbFile := range thumbFiles {
        s3Key := fmt.Sprintf(
            "thumbnails/%d/thumb_%d.jpg",
            videoUploadId, i+1,
        )

        _,err := utils.UploadToS3(thumbFile, s3Key)
        if err != nil {
            continue
        }

        cdnUrl := fmt.Sprintf("%s/%s", s3Base, s3Key)
        cdnUrls = append(cdnUrls, cdnUrl)
    }

    return cdnUrls, nil
}