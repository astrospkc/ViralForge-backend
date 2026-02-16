package middleware

import (
	"fmt"
	"strings"
	"time"
	"viralforge/src/env"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

func FetchUser() fiber.Handler{
	return func(c fiber.Ctx) error{
		envs:=env.NewEnv()
		authHeader := c.Get("Authorization")
		if authHeader==""||!strings.HasPrefix(authHeader, "Bearer "){
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication token missing or malformed",
			})
		}
		tknString := strings.TrimPrefix(authHeader, "Bearer ")

		secret := envs.JWT_KEY
		if secret == "" {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "JWT secret not configured",
			})
		}
		token, err := jwt.Parse(tknString, func(t *jwt.Token) (any, error) {
			// Check signing method
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secret), nil
		})
		// fmt.Println("check")
		// fmt.Println("token in middleware: ", token)
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token claims",
			})
		}
		if exp, ok := claims["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token expired",
			})
		}
		c.Locals("user", claims)
		// fmt.Println("user: ", claims) // or "id" / "email", whatever you stored

		return c.Next()
	}
}