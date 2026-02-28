package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

func FetchUserId(c fiber.Ctx) (string , error){
	
	var u_id string
	
	userIdInterface:= c.Locals("user")

	// fmt.Println("user interfacce: ", userIdInterface)
	claims, ok:=userIdInterface.(jwt.MapClaims)
	if !ok {
		return "",c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or missing  aud field",
		})
	}
	userIDInt := int64(claims["user_id"].(float64))

	u_id = strconv.Itoa(int(userIDInt))
	return u_id, nil
	
	
	
	
}