package service

import (
	"fmt"
	"os"
	"path/filepath"

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



func UploadThumbnails()