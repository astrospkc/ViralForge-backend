package handlers

import (
	"fmt"
	"strconv"
	"viralforge/src/connect"

	"github.com/gofiber/fiber/v3"
)

// GET /search/products - Search products by keyword (brand, name, description)
func SearchProducts() fiber.Handler {
	return func(c fiber.Ctx) error {
		query := c.Query("q")
		if query == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Query parameter 'q' is required",
			})
		}

		type ProductSearchResult struct {
			ID          int64   `json:"id"`
			Name        string  `json:"name"`
			Brand       string  `json:"brand"`
			Description string  `json:"description"`
			ShopCount   int64   `json:"shop_count"`
			MinPrice    float64 `json:"min_price"`
		}

		sqlQuery := `
			SELECT
				p.id,
				p.name,
				p.brand,
				p.description,
				COUNT(sp.id) AS shop_count,
				COALESCE(MIN(sp.price), 0) AS min_price
			FROM products p
			LEFT JOIN shop_products sp ON sp.product_id = p.id AND sp.status = 'ACTIVE'
			WHERE p.name ILIKE ? OR p.brand ILIKE ? OR p.description ILIKE ?
			GROUP BY p.id, p.name, p.brand, p.description
			LIMIT 50;
		`

		pattern := "%" + query + "%"
		rows, err := connect.Db.QueryContext(c.Context(), sqlQuery, pattern, pattern, pattern)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Search failed: %v", err),
			})
		}
		defer rows.Close()

		var results []ProductSearchResult
		for rows.Next() {
			var r ProductSearchResult
			if err := rows.Scan(&r.ID, &r.Name, &r.Brand, &r.Description, &r.ShopCount, &r.MinPrice); err == nil {
				results = append(results, r)
			}
		}

		if results == nil {
			results = []ProductSearchResult{}
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"data":    results,
		})
	}
}

// GET /search/nearby - Nearby product discovery (Section 4.3 in spec)
func SearchNearbyProducts() fiber.Handler {
	return func(c fiber.Ctx) error {
		latStr := c.Query("latitude")
		lngStr := c.Query("longitude")
		prodStr := c.Query("product_id")
		radStr := c.Query("radius")

		if latStr == "" || lngStr == "" || prodStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "latitude, longitude, and product_id query parameters are required",
			})
		}

		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid latitude"})
		}
		lng, err := strconv.ParseFloat(lngStr, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid longitude"})
		}
		productID, err := strconv.ParseInt(prodStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid product_id"})
		}

		radiusMeters := 5000.0
		if radStr != "" {
			if r, err := strconv.ParseFloat(radStr, 64); err == nil && r > 0 {
				radiusMeters = r
			}
		}

		type NearbyProductListing struct {
			ShopID         int64   `json:"shop_id"`
			ShopName       string  `json:"shop_name"`
			Address        string  `json:"address"`
			Price          float64 `json:"price"`
			Availability   string  `json:"availability"`
			DistanceMeters float64 `json:"distance_meters"`
		}

		// Exact query from Section 4.3
		query := `
			SELECT
				s.id,
				s.name,
				s.address,
				sp.price,
				sp.availability,
				ST_Distance(
					s.location,
					ST_SetSRID(
						ST_MakePoint(?, ?),
						4326
					)::geography
				) AS distance_meters
			FROM shops s
			JOIN shop_products sp
				ON sp.shop_id = s.id
			JOIN products p
				ON p.id = sp.product_id
			WHERE
				s.status = 'ACTIVE'
				AND s.verification_status = 'VERIFIED'
				AND sp.status = 'ACTIVE'
				AND p.id = ?
				AND ST_DWithin(
					s.location,
					ST_SetSRID(
						ST_MakePoint(?, ?),
						4326
					)::geography,
					?
				)
			ORDER BY distance_meters ASC;
		`

		rows, err := connect.Db.QueryContext(c.Context(), query, lng, lat, productID, lng, lat, radiusMeters)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to query nearby products: %v", err),
			})
		}
		defer rows.Close()

		var results []NearbyProductListing
		for rows.Next() {
			var r NearbyProductListing
			if err := rows.Scan(&r.ShopID, &r.ShopName, &r.Address, &r.Price, &r.Availability, &r.DistanceMeters); err == nil {
				results = append(results, r)
			}
		}

		if results == nil {
			results = []NearbyProductListing{}
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"data":    results,
		})
	}
}

// GET /discover/shops - Discover verified shops
func DiscoverShops() fiber.Handler {
	return func(c fiber.Ctx) error {
		city := c.Query("city")

		type DiscoverShopItem struct {
			ID          int64   `json:"id"`
			Name        string  `json:"name"`
			Description string  `json:"description"`
			Address     string  `json:"address"`
			City        string  `json:"city"`
			Latitude    float64 `json:"latitude"`
			Longitude   float64 `json:"longitude"`
			MerchantID  int64   `json:"merchant_id"`
		}

		query := connect.Db.NewSelect().
			TableExpr("shops AS s").
			ColumnExpr("s.id, s.name, s.description, s.address, s.city, s.merchant_id").
			ColumnExpr("ST_Y(s.location::geometry) AS latitude").
			ColumnExpr("ST_X(s.location::geometry) AS longitude").
			Where("s.status = 'ACTIVE' AND s.verification_status = 'VERIFIED'")

		if city != "" {
			query = query.Where("s.city ILIKE ?", "%"+city+"%")
		}

		query = query.Limit(50).Order("s.created_at DESC")

		var shops []DiscoverShopItem
		err := query.Scan(c.Context(), &shops)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to discover shops: %v", err),
			})
		}

		if shops == nil {
			shops = []DiscoverShopItem{}
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"data":    shops,
		})
	}
}

// GET /discover/videos - Discover public approved videos
func DiscoverVideos() fiber.Handler {
	return func(c fiber.Ctx) error {
		type VideoFeedItem struct {
			ID                int64   `json:"id"`
			UserID            int64   `json:"user_id"`
			MerchantID        *int64  `json:"merchant_id,omitempty"`
			ShopID            *int64  `json:"shop_id,omitempty"`
			ProductID         *int64  `json:"product_id,omitempty"`
			ShopProductID     *int64  `json:"shop_product_id,omitempty"`
			Title             string  `json:"title"`
			Description       string  `json:"description"`
			MasterCdnUrl      string  `json:"master_cdn_url"`
			SelectedThumbnail string  `json:"selected_thumbnail"`
			Duration          float64 `json:"video_duration"`
		}

		query := connect.Db.NewSelect().
			TableExpr("video_uploads AS vdu").
			ColumnExpr("vdu.id, vdu.user_id, vdu.merchant_id, vdu.shop_id, vdu.product_id, vdu.shop_product_id, vdu.title, vdu.description, vdu.master_cdn_url, vdu.selected_thumbnail, vdu.video_duration").
			Where("vdu.publish_status = 'published' AND vdu.is_deleted = false").
			Order("vdu.created_at DESC").
			Limit(50)

		var list []VideoFeedItem
		err := query.Scan(c.Context(), &list)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to discover videos: %v", err),
			})
		}

		if list == nil {
			list = []VideoFeedItem{}
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"data":    list,
		})
	}
}
