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
	auth.Post("/reset_code", handlers.SendCode())
	// auth.Post("/reset_password", handlers.ResetPassword())



	v1 := app.Group("/v1")
	// v1.Get("/presigned_url", handlers.GetPresignedUrl())
	//seach 
	v1.Get("/search",middleware.FetchUser(), handlers.SearchVideos())


	// Feed namespace
	feed := v1.Group("/feed")
	feed.Get("/",        handlers.GetAllPostVideosOfPlatform())   // GET /v1/feed
	feed.Get("/mine",middleware.FetchUser(), handlers.GetAllPostedVideosOfUser())      // GET /v1/feed/mine

	// Upload namespace  
	upload := v1.Group("/upload",middleware.FetchUser())
	upload.Post("/initiate", handlers.GetPresignedUrl())           // POST /v1/upload/initiate
	upload.Post("/commit",   handlers.CreateVideo())               // POST /v1/upload/commit

	// Videos namespace
	videos := v1.Group("/videos",middleware.FetchUser())
	videos.Get("/",              handlers.GetListOfVideoFiles())
	videos.Get("/:v_id",         handlers.GetTranscodedVideoDetails())
	videos.Get("/:v_id/status",  handlers.GetTranscodedVideoStatus())
	videos.Get("/download",		 handlers.GetDownloadUrl())
	videos.Put("/:v_id",         handlers.UpdateVideo())
	videos.Put("/:v_id/cdn",     handlers.UpdateCDN_Url())
	videos.Delete("/:v_id",      handlers.DeleteVideo())
	videos.Get("/:v_id/comments", handlers.GetTopLevelComments())



	// comment namespaces
	comments := videos.Group("/comments")
	comments.Post("/:v_id", handlers.CreateComment())
	comments.Get("/:parent_comment_id/replies", handlers.GetReplies())
	comments.Delete("/:comment_id", handlers.DeleteComment())
	comments.Patch("/:comment_id", handlers.UpdateComment())

	// video search 

	
}