package routes

import (
	"viralforge/src/handlers"
	"viralforge/src/middleware"

	"github.com/gofiber/fiber/v3"
)


func NormalRoutes(app *fiber.App){

	auth:=app.Group("/auth/v1")
	auth.Get("/", middleware.FetchUser(), handlers.GetUserFromId())
	auth.Post("/register",handlers.RegisterUser())
	auth.Post("/login", handlers.LoginUser())



	video:=app.Group("/video/v1", middleware.FetchUser())
	video.Get("/get_all_videos", handlers.GetListOfVideoFiles())
	video.Get("/get_download_url", handlers.GetDownloadUrl())
	video.Post("/get_presigned_url", handlers.GetPresignedUrl())
	video.Post("/create_video", handlers.AddVideoFileKeyToDB())
	video.Post("/transcode", handlers.VideoTranscode())
	video.Get("/details/:v_id", handlers.GetTranscodedVideoDetails())
	video.Get("/status/:v_id", handlers.GetTranscodedVideoStatus())
	video.Put("/cdnUpdate", handlers.UpdateCDN_Url())
	

	

}