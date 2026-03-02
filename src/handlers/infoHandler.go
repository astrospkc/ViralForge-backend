package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

func FetchUserId(c fiber.Ctx) (int64 , error){
	
	
	
	userIdInterface:= c.Locals("user")

	// fmt.Println("user interfacce: ", userIdInterface)
	claims, ok:=userIdInterface.(jwt.MapClaims)
	if !ok {
		return 0,c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or missing  aud field",
		})
	}
	userIDInt := int64(claims["user_id"].(float64))
	return userIDInt, nil
	
	
	
	
}