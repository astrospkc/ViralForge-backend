package handlers

import (
	"fmt"
	"viralforge/src/connect"
	"viralforge/src/models"

	"github.com/gofiber/fiber/v3"
)

type RegisterResponse struct{
	Message			string
	Data 			models.User
}


func RegisterUser() fiber.Handler{
	return func(c fiber.Ctx) error{
		var body models.User

		if err:= c.Bind().Body(&body); err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(RegisterResponse{
					Message: "null value",
					Data: models.User{},
				},
			)
		}

		// create user
		insertedUser,err := connect.Db.NewInsert().Model(&body).Returning("id, name, email, created_at, updated_at").Exec(c.Context())
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(RegisterResponse{
				Message: "could not insert user in db",
				Data:models.User{},
			})
		}
		fmt.Println("inserted user: ", insertedUser)


		return c.Status(fiber.StatusOK).JSON(RegisterResponse{
			Message:"successful fetching",
			Data:body,
		})
	}
}