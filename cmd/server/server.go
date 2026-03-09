package main

import (
	"viralforge/cmd/worker"
	"viralforge/src/connect"
	"viralforge/src/routes"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

func main(){

	// start http server

	go func(){
		app:= fiber.New()
	connect.PgConnect()

	app.Use(cors.New(cors.Config{
    AllowOrigins: []string{"http://localhost:5173"},
    AllowHeaders: []string{"Origin", "Content-Type", "Accept","Authorization"},
}))
	routes.NormalRoutes(app)
	app.Listen(":8081")
	}()

	// start worker server
	worker.StartWorkerServer()
	
}