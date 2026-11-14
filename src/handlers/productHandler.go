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

// POST /products - Create central product
func CreateProduct() fiber.Handler {
	return func(c fiber.Ctx) error {
		_, err := FetchUserId(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Unauthorized",
			})
		}

		var body struct {
			Name           string                 `json:"name"`
			Brand          string                 `json:"brand"`
			Description    string                 `json:"description"`
			CategoryID     *int64                 `json:"category_id"`
			Specifications map[string]interface{} `json:"specifications"`
		}

		if err := c.Bind().Body(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid request body",
			})
		}

		if strings.TrimSpace(body.Name) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Product name is required",
			})
		}

		product := &models.Product{
			Name:           body.Name,
			Brand:          body.Brand,
			Description:    body.Description,
			CategoryID:     body.CategoryID,
			Specifications: body.Specifications,
		}

		_, err = connect.Db.NewInsert().Model(product).Exec(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to create product: %v", err),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"success": true,
			"data":    product,
			"message": "Product created successfully",
		})
	}
}

// GET /products/:id - Product details
func GetProductDetails() fiber.Handler {
	return func(c fiber.Ctx) error {
		idParam := c.Params("id")
		productID, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid product id",
			})
		}

		var product models.Product
		err = connect.Db.NewSelect().
			Model(&product).
			Where("id = ?", productID).
			Scan(c.Context())

		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"message": "Product not found",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"data":    product,
		})
	}
}

// POST /shops/:id/products - Add product to shop catalogue
func AddProductToShop() fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, err := FetchUserId(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Unauthorized",
			})
		}

		shopIDParam := c.Params("id")
		shopID, err := strconv.ParseInt(shopIDParam, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid shop id",
			})
		}

		authorized, _, err := isUserAuthorizedForShop(c, shopID, userID, models.MerchantPermissionOwner, models.MerchantPermissionManager, models.MerchantPermissionInventoryManager)
		if err != nil || !authorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": "You are not authorized to add products to this shop",
			})
		}

		var body struct {
			ProductID    int64                    `json:"product_id"`
			Price        float64                  `json:"price"`
			Currency     string                   `json:"currency"`
			Availability models.StockAvailability `json:"availability"`
		}

		if err := c.Bind().Body(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid request body",
			})
		}

		if body.ProductID == 0 || body.Price <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Valid product_id and price are required",
			})
		}

		if body.Currency == "" {
			body.Currency = "INR"
		}
		if body.Availability == "" {
			body.Availability = models.AvailabilityAvailable
		}

		// Ensure product exists
		prodExists, err := connect.Db.NewSelect().Model((*models.Product)(nil)).Where("id = ?", body.ProductID).Exists(c.Context())
		if err != nil || !prodExists {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Product does not exist",
			})
		}

		shopProduct := &models.ShopProduct{
			ShopID:         shopID,
			ProductID:      body.ProductID,
			Price:          body.Price,
			Currency:       body.Currency,
			Availability:   body.Availability,
			StockUpdatedAt: time.Now(),
			Status:         models.ShopProductStatusActive,
		}

		_, err = connect.Db.NewInsert().
			Model(shopProduct).
			On("CONFLICT (shop_id, product_id) DO UPDATE").
			Set("price = EXCLUDED.price").
			Set("currency = EXCLUDED.currency").
			Set("availability = EXCLUDED.availability").
			Set("stock_updated_at = CURRENT_TIMESTAMP").
			Set("status = 'ACTIVE'").
			Set("updated_at = CURRENT_TIMESTAMP").
			Exec(c.Context())

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to add product to shop: %v", err),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"success": true,
			"data":    shopProduct,
			"message": "Product added to shop catalogue successfully",
		})
	}
}

// PATCH /shop-products/:id - Update price and stock
func UpdateShopProduct() fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, err := FetchUserId(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Unauthorized",
			})
		}

		idParam := c.Params("id")
		listingID, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid shop product id",
			})
		}

		var listing models.ShopProduct
		if err := connect.Db.NewSelect().Model(&listing).Where("id = ?", listingID).Scan(c.Context()); err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"message": "Shop product listing not found",
			})
		}

		authorized, _, err := isUserAuthorizedForShop(c, listing.ShopID, userID, models.MerchantPermissionOwner, models.MerchantPermissionManager, models.MerchantPermissionInventoryManager)
		if err != nil || !authorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": "You are not authorized to update this shop product",
			})
		}

		var body struct {
			Price        *float64                  `json:"price"`
			Currency     *string                   `json:"currency"`
			Availability *models.StockAvailability `json:"availability"`
			Status       *models.ShopProductStatus `json:"status"`
		}

		if err := c.Bind().Body(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid request body",
			})
		}

		updateQuery := connect.Db.NewUpdate().Model((*models.ShopProduct)(nil)).Where("id = ?", listingID)
		if body.Price != nil {
			updateQuery = updateQuery.Set("price = ?", *body.Price)
		}
		if body.Currency != nil {
			updateQuery = updateQuery.Set("currency = ?", *body.Currency)
		}
		if body.Availability != nil {
			updateQuery = updateQuery.Set("availability = ?", *body.Availability)
			updateQuery = updateQuery.Set("stock_updated_at = ?", time.Now())
		}
		if body.Status != nil {
			updateQuery = updateQuery.Set("status = ?", *body.Status)
		}
		updateQuery = updateQuery.Set("updated_at = ?", time.Now())

		_, err = updateQuery.Exec(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to update listing: %v", err),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"message": "Shop product listing updated successfully",
		})
	}
}

// GET /products/:id/shops - Shops selling product
func GetShopsSellingProduct() fiber.Handler {
	return func(c fiber.Ctx) error {
		idParam := c.Params("id")
		productID, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid product id",
			})
		}

		type ShopListingResult struct {
			ShopProductID  int64                    `json:"shop_product_id"`
			Price          float64                  `json:"price"`
			Currency       string                   `json:"currency"`
			Availability   models.StockAvailability `json:"availability"`
			StockUpdatedAt time.Time                `json:"stock_updated_at"`
			ShopID         int64                    `json:"shop_id"`
			ShopName       string                   `json:"shop_name"`
			Address        string                   `json:"address"`
			City           string                   `json:"city"`
			State          string                   `json:"state"`
			Pincode        string                   `json:"pincode"`
			Latitude       float64                  `json:"latitude"`
			Longitude      float64                  `json:"longitude"`
			MerchantID     int64                    `json:"merchant_id"`
		}

		var listings []ShopListingResult
		query := `
			SELECT
				sp.id AS shop_product_id,
				sp.price,
				sp.currency,
				sp.availability,
				sp.stock_updated_at,
				s.id AS shop_id,
				s.name AS shop_name,
				s.address,
				s.city,
				s.state,
				s.pincode,
				ST_Y(s.location::geometry) AS latitude,
				ST_X(s.location::geometry) AS longitude,
				s.merchant_id
			FROM shop_products sp
			JOIN shops s ON s.id = sp.shop_id
			WHERE sp.product_id = ?
			  AND sp.status = 'ACTIVE'
			  AND s.status = 'ACTIVE'
			ORDER BY sp.price ASC;
		`

		rows, err := connect.Db.QueryContext(c.Context(), query, productID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to query shops: %v", err),
			})
		}
		defer rows.Close()

		for rows.Next() {
			var r ShopListingResult
			if err := rows.Scan(
				&r.ShopProductID, &r.Price, &r.Currency, &r.Availability, &r.StockUpdatedAt,
				&r.ShopID, &r.ShopName, &r.Address, &r.City, &r.State, &r.Pincode,
				&r.Latitude, &r.Longitude, &r.MerchantID,
			); err == nil {
				listings = append(listings, r)
			}
		}

		if listings == nil {
			listings = []ShopListingResult{}
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"data":    listings,
		})
	}
}
