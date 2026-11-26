package routes

import (
	"viralforge/src/handlers"
	"viralforge/src/middleware"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NormalRoutes(app *fiber.App) {
	app.Get("/", handlers.Home())

	// Prometheus Metrics Endpoint (Scraped by Prometheus on :9090)
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	auth := app.Group("/auth/v1")
	auth.Post("/send_email_otp", handlers.SendEmailWithRedisOTP())
	auth.Get("/", middleware.FetchUser(), handlers.GetUserFromId())
	auth.Post("/register", handlers.RegisterUser())
	auth.Post("/login", handlers.LoginUser())
	auth.Post("/reset_code", handlers.SendCode())
	auth.Post("/verify_code", handlers.VerifyOTP())
	auth.Post("/reset_password", handlers.ResetPassword())

	v1 := app.Group("/v1")
	v1.Get("/search", middleware.FetchUser(), handlers.SearchVideos())

	// Feed namespace
	feed := v1.Group("/feed")
	feed.Get("/", handlers.GetAllPostVideosOfPlatform())                           // GET /v1/feed
	feed.Get("/mine", middleware.FetchUser(), handlers.GetAllPostedVideosOfUser()) // GET /v1/feed/mine

	// Upload namespace
	upload := v1.Group("/upload", middleware.FetchUser())
	upload.Post("/initiate", handlers.GetPresignedUrl()) // POST /v1/upload/initiate
	upload.Post("/commit", handlers.CreateVideo())       // POST /v1/upload/commit

	// Videos namespace
	videos := v1.Group("/videos", middleware.FetchUser())
	videos.Get("/", handlers.GetListOfVideoFiles())
	videos.Get("/download", handlers.GetDownloadUrl())
	videos.Get("/:v_id", handlers.GetVideoDetailsWithRelations())
	videos.Get("/:v_id/status", handlers.GetTranscodedVideoStatus())
	videos.Put("/:v_id", handlers.UpdateVideo())
	videos.Put("/:v_id/cdn", handlers.UpdateCDN_Url())
	videos.Delete("/:v_id", handlers.DeleteVideo())
	videos.Get("/:v_id/related-shops", handlers.GetVideoRelatedShops())
	videos.Get("/:v_id/related-products", handlers.GetVideoRelatedProducts())
	videos.Get("/:v_id/comments", handlers.GetTopLevelComments())

	// comment namespaces
	comments := videos.Group("/comments")
	comments.Post("/:v_id", handlers.CreateComment())
	comments.Get("/:parent_comment_id/replies", handlers.GetReplies())
	comments.Delete("/:comment_id", handlers.DeleteComment())
	comments.Patch("/:comment_id", handlers.UpdateComment())

	// Merchants
	merchants := v1.Group("/merchants")
	merchants.Post("/", middleware.FetchUser(), handlers.CreateMerchantProfile())
	merchants.Get("/:id", handlers.GetMerchantProfile())
	merchants.Patch("/:id", middleware.FetchUser(), handlers.UpdateMerchantProfile())
	merchants.Get("/:id/shops", handlers.GetMerchantShops())
	merchants.Get("/:id/videos", handlers.GetMerchantVideos())

	// Shops
	shops := v1.Group("/shops")
	shops.Post("/", middleware.FetchUser(), handlers.CreateShop())
	shops.Get("/nearby", handlers.GetNearbyShops())
	shops.Get("/:id", handlers.GetShopDetails())
	shops.Patch("/:id", middleware.FetchUser(), handlers.UpdateShop())
	shops.Get("/:id/products", handlers.GetShopProducts())
	shops.Get("/:id/videos", handlers.GetShopVideos())
	shops.Post("/:id/products", middleware.FetchUser(), handlers.AddProductToShop())

	// Products
	products := v1.Group("/products")
	products.Post("/", middleware.FetchUser(), handlers.CreateProduct())
	products.Get("/:id", handlers.GetProductDetails())
	products.Get("/:id/shops", handlers.GetShopsSellingProduct())

	// Shop Products listing
	v1.Patch("/shop-products/:id", middleware.FetchUser(), handlers.UpdateShopProduct())

	// Feed
	v1.Get("/feed/videos", handlers.DiscoverVideos())

	// Discovery & Search
	v1.Get("/search/products", handlers.SearchProducts())
	v1.Get("/search/nearby", handlers.SearchNearbyProducts())
	v1.Get("/discover/shops", handlers.DiscoverShops())
	v1.Get("/discover/videos", handlers.DiscoverVideos())

	// Customer saved locations
	savedLocations := v1.Group("/locations/saved", middleware.FetchUser())
	savedLocations.Post("/", handlers.SaveUserLocation())
	savedLocations.Get("/", handlers.GetUserSavedLocations())
	savedLocations.Delete("/:id", handlers.DeleteUserSavedLocation())
}
