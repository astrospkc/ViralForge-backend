package handlers

import (
	"fmt"
	"time"
	"viralforge/src/connect"
	"viralforge/src/env"
	"viralforge/src/models"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type RegisterResponse struct{
	Message			string
	Data 			models.User
	Token  			string
}

var Jwt_key string

type Claims struct{
	UserId int64	`json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateToken(userId int64) (string, error){
	envs:=env.NewEnv()
	Jwt_key = envs.JWT_KEY
	expirationTime:= time.Now().Add(5*time.Hour)
	 claims := &Claims{
        UserId: userId,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(expirationTime),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Subject:   "user authentication",
        },
    }
	token:=jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	fmt.Println("token: ", token)
	tokenString, err := token.SignedString([]byte(Jwt_key))
    if err != nil {
        return "", fmt.Errorf("failed to sign token: %w", err)
    }
    
    fmt.Printf("Final JWT token: %s\n", tokenString)
    return tokenString, nil
}

func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
    return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
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
		// 1. first check if user exists
		userEmail:=body.Email 
		exists,err := connect.Db.NewSelect().Model((*models.User)(nil)).Where("email = ?",userEmail).Exists(c.Context())
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(RegisterResponse{
				Message: "Failed to check user",
				Data:models.User{},
			})
		}
		if exists{
			return c.Status(fiber.StatusBadRequest).JSON(RegisterResponse{
				Message: "user already exist",
			})
		}
		
		

		hashPassword,err:= HashPassword(body.Password)
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(RegisterResponse{
				Message: "Failed to hash password",
				Data:models.User{},
			})
		}

		user:=&models.User{
			Name:body.Name,
			Email:body.Email,
			Password: hashPassword,
		}

		// create user
		_,err= connect.Db.NewInsert().Model(user).Returning("id, name, email, created_at, updated_at").Exec(c.Context())
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(RegisterResponse{
				Message: "could not insert user in db",
				Data:models.User{},
			})
		}

		token, err:= GenerateToken(user.ID)
		if err!=nil{
			_, err = connect.Db.NewDelete().
				Model((*models.User)(nil)).
				Where("email = ?", userEmail).
				Exec(c.Context())
			return c.Status(fiber.StatusBadRequest).JSON(RegisterResponse{
				Message: "failed to generate token",
			})
		}

		
	
		return c.Status(fiber.StatusOK).JSON(RegisterResponse{
			Message:"successful fetching",
			Data:*user,
			Token:token,
		})
	}
}

