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



func UploadThumbnails()