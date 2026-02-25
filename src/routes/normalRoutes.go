package routes

import (
	"viralforge/src/handlers"

	"github.com/gofiber/fiber/v3"
)


func NormalRoutes(app *fiber.App){

	auth:=app.Group("/auth/v1")
	auth.Post("/register",handlers.RegisterUser())
	auth.Post("/login", handlers.LoginUser())

	video:=app.Group("video/v1")
	video.Post("/get_presigned_url", handlers.GetPresignedUrl())

}