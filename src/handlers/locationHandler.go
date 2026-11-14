package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"viralforge/src/connect"
	"viralforge/src/models"

	"github.com/gofiber/fiber/v3"
)

// POST /locations/saved - Save customer location (Home, Work, etc.)
func SaveUserLocation() fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, err := FetchUserId(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Unauthorized",
			})
		}

		var body struct {
			Label     string  `json:"label"`
			Address   string  `json:"address"`
			City      string  `json:"city"`
			Pincode   string  `json:"pincode"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}

		if err := c.Bind().Body(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid request body",
			})
		}

		if body.Latitude < -90 || body.Latitude > 90 || body.Longitude < -180 || body.Longitude > 180 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid latitude (-90 to 90) or longitude (-180 to 180)",
			})
		}

		if strings.TrimSpace(body.Label) == "" {
			body.Label = "Saved Location"
		}

		var insertedID int64
		query := `
			INSERT INTO user_saved_locations (user_id, label, address, city, pincode, location, created_at)
			VALUES (?, ?, ?, ?, ?, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography, CURRENT_TIMESTAMP)
			RETURNING id;
		`
		err = connect.Db.QueryRowContext(c.Context(), query,
			userID, body.Label, body.Address, body.City, body.Pincode, body.Longitude, body.Latitude,
		).Scan(&insertedID)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to save location: %v", err),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"id":        insertedID,
				"user_id":   userID,
				"label":     body.Label,
				"address":   body.Address,
				"city":      body.City,
				"pincode":   body.Pincode,
				"latitude":  body.Latitude,
				"longitude": body.Longitude,
			},
			"message": "Location saved successfully",
		})
	}
}

// GET /locations/saved - List customer's saved locations
func GetUserSavedLocations() fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, err := FetchUserId(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Unauthorized",
			})
		}

		type SavedLoc struct {
			ID        int64   `json:"id"`
			UserID    int64   `json:"user_id"`
			Label     string  `json:"label"`
			Address   string  `json:"address"`
			City      string  `json:"city"`
			Pincode   string  `json:"pincode"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}

		query := `
			SELECT
				id, user_id, label, address, city, pincode,
				ST_Y(location::geometry) AS latitude,
				ST_X(location::geometry) AS longitude
			FROM user_saved_locations
			WHERE user_id = ?
			ORDER BY created_at DESC;
		`

		rows, err := connect.Db.QueryContext(c.Context(), query, userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to fetch saved locations: %v", err),
			})
		}
		defer rows.Close()

		var results []SavedLoc
		for rows.Next() {
			var loc SavedLoc
			if err := rows.Scan(&loc.ID, &loc.UserID, &loc.Label, &loc.Address, &loc.City, &loc.Pincode, &loc.Latitude, &loc.Longitude); err == nil {
				results = append(results, loc)
			}
		}

		if results == nil {
			results = []SavedLoc{}
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"data":    results,
		})
	}
}

// DELETE /locations/saved/:id - Delete a saved location
func DeleteUserSavedLocation() fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, err := FetchUserId(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"message": "Unauthorized",
			})
		}

		idParam := c.Params("id")
		locID, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Invalid location id",
			})
		}

		res, err := connect.Db.NewDelete().
			Model((*models.UserSavedLocation)(nil)).
			Where("id = ? AND user_id = ?", locID, userID).
			Exec(c.Context())

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": fmt.Sprintf("Failed to delete saved location: %v", err),
			})
		}

		rows, _ := res.RowsAffected()
		if rows == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"message": "Saved location not found",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"message": "Saved location deleted successfully",
		})
	}
}
