package handlers

import (
	"viralforge/src/connect"
	"viralforge/src/models"

	"github.com/gofiber/fiber/v3"
)



type PostVideoResponse struct{
	Data   *models.Post 
	Success bool 
	Code    int
}


// User sees ONE video post
//   ↓
// behind the scenes = multiple quality rows
//   video_uploads row (1)     ← the "post" — title, desc, thumbnail
//   video_qualities rows (3)  ← 1080p, 720p, 480p — hidden from user

// User never picks resolution when posting
// Player auto-picks best quality based on connection
// User can manually change in settings/player if they want
func PostVideo() fiber.Handler {
	return func(c fiber.Ctx)error{
		// get the video id
		// get the user id
		// write title , description and tags 
		v_id:=c.Query("v_id")
		user_id, err:= FetchUserId(c)
		if err!=nil{
			return c.Status(fiber.StatusInternalServerError).JSON(PostVideoResponse{
				Success: false,
				Code: 500,
			})
		}

		var body struct{
			Title string
			Description string
			Tags        string
		}

		if err= c.Bind().Body(&body); err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(PostVideoResponse{
				Success: false,
				Code:400,
			})
		}

		var video models.VideoQuality
		err= connect.Db.NewSelect().Model()
		return c.Status(fiber.StatusBadRequest).JSON(PostVideoResponse{
			Success: false,
			Code:400,
		})
	}
}