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

// Check if user is owner or authorized member of merchant
func isUserAuthorizedForMerchant(c fiber.Ctx, merchantID int64, userID int64, requiredPerms ...models.MerchantPermission) (bool, error) {
	var profile models.MerchantProfile
	err := connect.Db.NewSelect().
		Model(&profile).
		Where("id = ?", merchantID).
		Scan(c.Context())
	if err != nil {
		return false, err
	}

	if profile.OwnerUserID == userID {
		return true, nil
	}

	if len(requiredPerms) == 0 {
		requiredPerms = []models.MerchantPermission{
			models.MerchantPermissionOwner,
			models.MerchantPermissionManager,
			models.MerchantPermissionContentManager,
			models.MerchantPermissionInventoryManager,
		}
	}

	var member models.MerchantMember
	q := connect.Db.NewSelect().
		Model(&member).
		Where("merchant_id = ? AND user_id = ? AND status = ?", merchantID, userID, models.MemberStatusActive)

	permStrings := make([]string, len(requiredPerms))
	for i, p := range requiredPerms {
		permStrings[i] = string(p)
	}
	q = q.Where("permission IN (?)", permStrings)

	exists, err := q.Exists(c.Context())
	if err != nil {
		return false, err
	}
	return exists, nil
}

// POST /merchants - Create merchant profile
func CreateMerchantProfile() fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, err := FetchUserId(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Unauthorized",
			})
		}

		var body struct {
			BusinessName        string              `json:"business_name"`
			BusinessDescription string              `json:"business_description"`
			BusinessType        models.BusinessType `json:"business_type"`
		}

		if err := c.Bind().Body(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid request body",
			})
		}

		if strings.TrimSpace(body.BusinessName) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Business name is required",
			})
		}

		if body.BusinessType == "" {
			body.BusinessType = models.BusinessTypeRetail
		}

		merchant := &models.MerchantProfile{
			OwnerUserID:         userID,
			BusinessName:        body.BusinessName,
			BusinessDescription: body.BusinessDescription,
			BusinessType:        body.BusinessType,
			VerificationStatus:  models.VerificationPending,
		}

		_, err = connect.Db.NewInsert().Model(merchant).Exec(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to create merchant profile: %v", err),
			})
		}

		// Also associate or assign MERCHANT role if roles table has it
		var merchantRole models.Role
		if err := connect.Db.NewSelect().Model(&merchantRole).Where("name = ?", models.RoleMerchant).Scan(c.Context()); err == nil {
			userRole := &models.UserRole{
				UserID: userID,
				RoleID: merchantRole.ID,
			}
			_, _ = connect.Db.NewInsert().Model(userRole).Ignore().Exec(c.Context())
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"success": true,
			"data":    merchant,
			"message": "Merchant profile created successfully",
		})
	}
}

// GET /merchants/:id - Get merchant profile
func GetMerchantProfile() fiber.Handler {
	return func(c fiber.Ctx) error {
		idParam := c.Params("id")
		merchantID, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid merchant id",
			})
		}

		var merchant models.MerchantProfile
		err = connect.Db.NewSelect().
			Model(&merchant).
			Relation("Owner").
			Where("mp.id = ?", merchantID).
			Scan(c.Context())
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"message": "Merchant profile not found",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"data":    merchant,
		})
	}
}

// PATCH /merchants/:id - Update merchant profile
func UpdateMerchantProfile() fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, err := FetchUserId(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Unauthorized",
			})
		}

		idParam := c.Params("id")
		merchantID, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid merchant id",
			})
		}

		authorized, err := isUserAuthorizedForMerchant(c, merchantID, userID, models.MerchantPermissionOwner, models.MerchantPermissionManager)
		if err != nil || !authorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"message": "You are not authorized to update this merchant",
			})
		}

		var body struct {
			BusinessName        *string              `json:"business_name"`
			BusinessDescription *string              `json:"business_description"`
			BusinessType        *models.BusinessType `json:"business_type"`
		}
		if err := c.Bind().Body(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid request body",
			})
		}

		updateQuery := connect.Db.NewUpdate().Model((*models.MerchantProfile)(nil)).Where("id = ?", merchantID)
		if body.BusinessName != nil {
			updateQuery = updateQuery.Set("business_name = ?", *body.BusinessName)
		}
		if body.BusinessDescription != nil {
			updateQuery = updateQuery.Set("business_description = ?", *body.BusinessDescription)
		}
		if body.BusinessType != nil {
			updateQuery = updateQuery.Set("business_type = ?", *body.BusinessType)
		}
		updateQuery = updateQuery.Set("updated_at = ?", time.Now())

		_, err = updateQuery.Exec(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to update merchant: %v", err),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"message": "Merchant profile updated successfully",
		})
	}
}

// GET /merchants/:id/shops - List shops of a merchant
func GetMerchantShops() fiber.Handler {
	return func(c fiber.Ctx) error {
		idParam := c.Params("id")
		merchantID, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid merchant id",
			})
		}

		type ShopWithCoords struct {
			models.Shop
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}

		var shops []ShopWithCoords
		err = connect.Db.NewSelect().
			TableExpr("shops AS s").
			ColumnExpr("s.*").
			ColumnExpr("ST_Y(s.location::geometry) AS latitude").
			ColumnExpr("ST_X(s.location::geometry) AS longitude").
			Where("s.merchant_id = ?", merchantID).
			Scan(c.Context(), &shops)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to fetch shops: %v", err),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"data":    shops,
		})
	}
}

// GET /merchants/:id/videos - List videos of a merchant
func GetMerchantVideos() fiber.Handler {
	return func(c fiber.Ctx) error {
		idParam := c.Params("id")
		merchantID, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid merchant id",
			})
		}

		var videos []models.VideoUpload
		err = connect.Db.NewSelect().
			Model(&videos).
			Where("merchant_id = ? AND publish_status = 'published' AND is_deleted = false", merchantID).
			Order("created_at DESC").
			Scan(c.Context())

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to fetch videos: %v", err),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"data":    videos,
		})
	}
}
