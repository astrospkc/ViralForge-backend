package handlers

import (
	"viralforge/src/connect"
	"viralforge/src/models"

	"github.com/gofiber/fiber/v3"
)


func GetThumbnailOptions() fiber.Handler {
    return func(c fiber.Ctx) error {
        videoId := c.Params("videoId")

        var video models.VideoUpload
        err := connect.Db.NewSelect().
            Model(&video).
            Where("id = ?", videoId).
            Scan(c.Context())

        if err != nil {
            return c.Status(500).JSON(fiber.Map{"success": false})
        }

        return c.Status(200).JSON(fiber.Map{
            "success":       true,
            "thumbnails":    video.Thumbnails,   // all 5 options
            "selected":      video.SelectedThumbnail,   // current pick
        })
    }
}

func SetSelectedThumbnail() fiber.Handler {
    return func(c fiber.Ctx) error {
        videoId := c.Params("videoId")

        var body struct {
            ThumbnailUrl string `json:"thumbnail_url"`
        }
        c.Bind().Body(&body)

        connect.Db.NewUpdate().
            TableExpr("video_uploads").
            Set("selected_thumb = ?", body.ThumbnailUrl).
            Where("id = ?", videoId).
            Exec(c.Context())

        return c.Status(200).JSON(fiber.Map{"success": true})
    }
}