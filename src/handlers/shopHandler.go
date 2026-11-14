package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"viralforge/src/connect"
	"viralforge/src/models"

	"github.com/gofiber/fiber/v3"
)

// Helper: check if shop belongs to a merchant where user is authorized
func isUserAuthorizedForShop(c fiber.Ctx, shopID int64, userID int64, requiredPerms ...models.MerchantPermission) (bool, *models.Shop, error) {
	var shop models.Shop
	err := connect.Db.NewSelect().
		Model(&shop).
		Where("id = ?", shopID).
		Scan(c.Context())
	if err != nil {
		return false, nil, err
	}

	authorized, err := isUserAuthorizedForMerchant(c, shop.MerchantID, userID, requiredPerms...)
	if err != nil || !authorized {
		return false, &shop, err
	}
	return true, &shop, nil
}

// POST /shops - Create a shop
func CreateShop() fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, err := FetchUserId(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Unauthorized",
			})
		}

		var body struct {
			MerchantID  int64   `json:"merchant_id"`
			Name        string  `json:"name"`
			Description string  `json:"description"`
			Phone       string  `json:"phone"`
			Address     string  `json:"address"`
			City        string  `json:"city"`
			State       string  `json:"state"`
			Pincode     string  `json:"pincode"`
			Country     string  `json:"country"`
			Latitude    float64 `json:"latitude"`
			Longitude   float64 `json:"longitude"`
		}

		if err := c.Bind().Body(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid request body",
			})
		}

		if body.MerchantID == 0 || strings.TrimSpace(body.Name) == "" || strings.TrimSpace(body.Address) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "merchant_id, name, and address are required",
			})
		}

		// Validation on coordinates
		if body.Latitude < -90 || body.Latitude > 90 || body.Longitude < -180 || body.Longitude > 180 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid latitude (-90 to 90) or longitude (-180 to 180)",
			})
		}

		// Authorize: user must be owner or manager of merchant
		authorized, err := isUserAuthorizedForMerchant(c, body.MerchantID, userID, models.MerchantPermissionOwner, models.MerchantPermissionManager)
		if err != nil || !authorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": "You are not authorized to create a shop for this merchant",
			})
		}

		if body.Country == "" {
			body.Country = "India"
		}

		var insertedID int64
		query := `
			INSERT INTO shops (merchant_id, name, description, phone, address, city, state, pincode, country, location, status, verification_status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography, 'ACTIVE', 'PENDING', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			RETURNING id;
		`
		err = connect.Db.QueryRowContext(c.Context(), query,
			body.MerchantID, body.Name, body.Description, body.Phone, body.Address, body.City, body.State, body.Pincode, body.Country, body.Longitude, body.Latitude,
		).Scan(&insertedID)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to create shop: %v", err),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"id":                  insertedID,
				"merchant_id":         body.MerchantID,
				"name":                body.Name,
				"description":         body.Description,
				"phone":               body.Phone,
				"address":             body.Address,
				"city":                body.City,
				"state":               body.State,
				"pincode":             body.Pincode,
				"country":             body.Country,
				"latitude":            body.Latitude,
				"longitude":           body.Longitude,
				"status":              models.ShopStatusActive,
				"verification_status": models.VerificationPending,
			},
			"message": "Shop created successfully",
		})
	}
}

// GET /shops/:id - Get shop details
func GetShopDetails() fiber.Handler {
	return func(c fiber.Ctx) error {
		idParam := c.Params("id")
		shopID, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid shop id",
			})
		}

		type ShopDetail struct {
			models.Shop
			Latitude  float64                 `json:"latitude"`
			Longitude float64                 `json:"longitude"`
			Merchant  *models.MerchantProfile `bun:"rel:belongs-to,join:merchant_id=id" json:"merchant,omitempty"`
		}

		var detail ShopDetail
		err = connect.Db.NewSelect().
			TableExpr("shops AS s").
			ColumnExpr("s.*").
			ColumnExpr("ST_Y(s.location::geometry) AS latitude").
			ColumnExpr("ST_X(s.location::geometry) AS longitude").
			Relation("Merchant").
			Where("s.id = ?", shopID).
			Scan(c.Context(), &detail)

		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"message": "Shop not found",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"data":    detail,
		})
	}
}

// PATCH /shops/:id - Update shop
func UpdateShop() fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, err := FetchUserId(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Unauthorized",
			})
		}

		idParam := c.Params("id")
		shopID, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid shop id",
			})
		}

		authorized, _, err := isUserAuthorizedForShop(c, shopID, userID, models.MerchantPermissionOwner, models.MerchantPermissionManager)
		if err != nil || !authorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": "You are not authorized to update this shop",
			})
		}

		var body struct {
			Name        *string            `json:"name"`
			Description *string            `json:"description"`
			Phone       *string            `json:"phone"`
			Address     *string            `json:"address"`
			City        *string            `json:"city"`
			State       *string            `json:"state"`
			Pincode     *string            `json:"pincode"`
			Country     *string            `json:"country"`
			Latitude    *float64           `json:"latitude"`
			Longitude   *float64           `json:"longitude"`
			Status      *models.ShopStatus `json:"status"`
		}

		if err := c.Bind().Body(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid request body",
			})
		}

		updateQuery := connect.Db.NewUpdate().TableExpr("shops").Where("id = ?", shopID)
		if body.Name != nil {
			updateQuery = updateQuery.Set("name = ?", *body.Name)
		}
		if body.Description != nil {
			updateQuery = updateQuery.Set("description = ?", *body.Description)
		}
		if body.Phone != nil {
			updateQuery = updateQuery.Set("phone = ?", *body.Phone)
		}
		if body.Address != nil {
			updateQuery = updateQuery.Set("address = ?", *body.Address)
		}
		if body.City != nil {
			updateQuery = updateQuery.Set("city = ?", *body.City)
		}
		if body.State != nil {
			updateQuery = updateQuery.Set("state = ?", *body.State)
		}
		if body.Pincode != nil {
			updateQuery = updateQuery.Set("pincode = ?", *body.Pincode)
		}
		if body.Country != nil {
			updateQuery = updateQuery.Set("country = ?", *body.Country)
		}
		if body.Status != nil {
			updateQuery = updateQuery.Set("status = ?", *body.Status)
		}
		if body.Latitude != nil && body.Longitude != nil {
			if *body.Latitude < -90 || *body.Latitude > 90 || *body.Longitude < -180 || *body.Longitude > 180 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"success": false,
					"message": "Invalid latitude (-90 to 90) or longitude (-180 to 180)",
				})
			}
			updateQuery = updateQuery.Set("location = ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography", *body.Longitude, *body.Latitude)
		}
		updateQuery = updateQuery.Set("updated_at = ?", time.Now())

		_, err = updateQuery.Exec(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to update shop: %v", err),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"message": "Shop updated successfully",
		})
	}
}

// GET /shops/:id/products - Get shop catalogue
func GetShopProducts() fiber.Handler {
	return func(c fiber.Ctx) error {
		idParam := c.Params("id")
		shopID, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid shop id",
			})
		}

		var listings []models.ShopProduct
		err = connect.Db.NewSelect().
			Model(&listings).
			Relation("Product").
			Where("sp.shop_id = ? AND sp.status = 'ACTIVE'", shopID).
			Order("sp.created_at DESC").
			Scan(c.Context())

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to fetch shop catalogue: %v", err),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"data":    listings,
		})
	}
}

// GET /shops/:id/videos - Get videos linked to a shop
func GetShopVideos() fiber.Handler {
	return func(c fiber.Ctx) error {
		idParam := c.Params("id")
		shopID, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid shop id",
			})
		}

		var videos []models.VideoUpload
		err = connect.Db.NewSelect().
			Model(&videos).
			Where("shop_id = ? AND publish_status = 'published' AND is_deleted = false", shopID).
			Order("created_at DESC").
			Scan(c.Context())

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to fetch shop videos: %v", err),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"data":    videos,
		})
	}
}

// GET /shops/nearby - Discover nearby shops (Section 4.3 & Section 8)
func GetNearbyShops() fiber.Handler {
	return func(c fiber.Ctx) error {
		latStr := c.Query("latitude")
		lngStr := c.Query("longitude")
		radStr := c.Query("radius") // in meters or km, defaults to 5000 meters

		if latStr == "" || lngStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "latitude and longitude query parameters are required",
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

		radiusMeters := 5000.0
		if radStr != "" {
			if r, err := strconv.ParseFloat(radStr, 64); err == nil && r > 0 {
				radiusMeters = r
			}
		}

		type NearbyShopResult struct {
			ID                 int64   `json:"id"`
			MerchantID         int64   `json:"merchant_id"`
			Name               string  `json:"name"`
			Description        string  `json:"description"`
			Phone              string  `json:"phone"`
			Address            string  `json:"address"`
			City               string  `json:"city"`
			State              string  `json:"state"`
			Pincode            string  `json:"pincode"`
			Country            string  `json:"country"`
			Latitude           float64 `json:"latitude"`
			Longitude          float64 `json:"longitude"`
			VerificationStatus string  `json:"verification_status"`
			DistanceMeters     float64 `json:"distance_meters"`
		}

		var results []NearbyShopResult
		query := `
			SELECT
				s.id,
				s.merchant_id,
				s.name,
				s.description,
				s.phone,
				s.address,
				s.city,
				s.state,
				s.pincode,
				s.country,
				ST_Y(s.location::geometry) AS latitude,
				ST_X(s.location::geometry) AS longitude,
				s.verification_status,
				ST_Distance(
					s.location,
					ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography
				) AS distance_meters
			FROM shops s
			WHERE
				s.status = 'ACTIVE'
				AND s.verification_status = 'VERIFIED'
				AND ST_DWithin(
					s.location,
					ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography,
					?
				)
			ORDER BY distance_meters ASC;
		`

		rows, err := connect.Db.QueryContext(c.Context(), query, lng, lat, lng, lat, radiusMeters)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to query nearby shops: %v", err),
			})
		}
		defer rows.Close()

		for rows.Next() {
			var r NearbyShopResult
			if err := rows.Scan(
				&r.ID, &r.MerchantID, &r.Name, &r.Description, &r.Phone, &r.Address,
				&r.City, &r.State, &r.Pincode, &r.Country, &r.Latitude, &r.Longitude,
				&r.VerificationStatus, &r.DistanceMeters,
			); err == nil {
				results = append(results, r)
			}
		}

		if results == nil {
			results = []NearbyShopResult{}
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"data":    results,
		})
	}
}
