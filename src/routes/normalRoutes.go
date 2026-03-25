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



	v1 := app.Group("/v1", middleware.FetchUser())

	// Feed namespace
	feed := v1.Group("/feed")
	feed.Get("/",        handlers.GetAllPostVideosOfPlatform())   // GET /v1/feed
	feed.Get("/mine",    handlers.GetAllPostedVideosOfUser())      // GET /v1/feed/mine

	// Upload namespace  
	upload := v1.Group("/upload")
	upload.Post("/initiate", handlers.GetPresignedUrl())           // POST /v1/upload/initiate
	upload.Post("/commit",   handlers.CreateVideo())               // POST /v1/upload/commit

	// Videos namespace
	videos := v1.Group("/videos")
	videos.Get("/",              handlers.GetListOfVideoFiles())
	videos.Get("/:v_id",         handlers.GetTranscodedVideoDetails())
	videos.Get("/:v_id/status",  handlers.GetTranscodedVideoStatus())
	videos.Get("/download",		 handlers.GetDownloadUrl())
	videos.Put("/:v_id",         handlers.UpdateVideo())
	videos.Put("/:v_id/cdn",     handlers.UpdateCDN_Url())
	videos.Delete("/:v_id",      handlers.DeleteVideo())
	
}