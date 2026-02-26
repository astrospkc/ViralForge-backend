package handlers

import (
	"fmt"
	"strconv"
	"time"
	"viralforge/src/connect"
	"viralforge/src/env"
	"viralforge/src/models"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthResponse struct{
	Message			string
	Data 			UserResponse
	Token  			string
	Success 		bool
}

type UserResponse struct{
	ID   	int64 
	Name    string 
	Email   string
	CreatedAt time.Time
	UpdatedAt time.Time
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
            Subject:   "user_authentication",
        },
    }
	token:=jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	tokenString, err := token.SignedString([]byte(Jwt_key))
    if err != nil {
        return "", fmt.Errorf("failed to sign token: %w", err)
    }
    
    
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
			return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
					Message: "null value",
					Data: UserResponse{},
				},
			)
		}
		// 1. first check if user exists
		fmt.Println("user provided details: ", body)
		userEmail:=body.Email 
		exists,err := connect.Db.NewSelect().Model((*models.User)(nil)).Where("email = ?",userEmail).Exists(c.Context())
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
				Message: "Failed to check user",
				Data:UserResponse{},
			})
		}
		fmt.Println("user exists; ", exists)
		if exists{
			return c.Status(fiber.StatusConflict).JSON(AuthResponse{
				Message: "user already exist",
			})
		}
		
		

		hashPassword,err:= HashPassword(body.Password)
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
				Message: "Failed to hash password",
				Data:UserResponse{},
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
			return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
				Message: "could not insert user in db",
				Data:UserResponse{},
			})
		}

		token, err:= GenerateToken(user.ID)
		if err!=nil{
			_, err = connect.Db.NewDelete().
				Model((*models.User)(nil)).
				Where("email = ?", userEmail).
				Exec(c.Context())
			return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
				Message: "failed to generate token",
			})
		}

		resp_user:=UserResponse{
			ID: user.ID,
			Name: user.Name,
			Email: user.Email,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		}

		
	
		return c.Status(fiber.StatusOK).JSON(AuthResponse{
			Message:"successful fetching",
			Data:resp_user,
			Token:token,
			Success: true,
		})
	}
}



func LoginUser() fiber.Handler{
	return func(c fiber.Ctx) error{
		
		var body struct{
			Email	string 
			Password string
		}

		if err := c.Bind().Body(&body); err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
				Message: "Please provide email or password or both",
			})
		}

		// verify password and return the result with token
		var user models.User
		err := connect.Db.NewSelect().Model((&user)).Where("email = ?",body.Email).Scan(c.Context())
		if err!=nil{
			return c.Status(fiber.StatusNotFound).JSON(AuthResponse{
				Message: "failed to fetch",
			})
		}
		

		isPasswordCorrect := CheckPasswordHash(body.Password, user.Password)
		if !isPasswordCorrect{
			return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
				Message:"Password is incorrect",
				Success:false,
			})
		}

		tokenString, err := GenerateToken(user.ID)
		resp_user:=UserResponse{
			ID:user.ID,
			Name: user.Name,
			Email:user.Email,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,

		}

		return c.Status(fiber.StatusOK).JSON(AuthResponse{
			Message: "logged in successfully",
			Success: true,
			Data:  resp_user,
			Token: tokenString,
		})
		
	}
}


type GetUserResponse struct{
	Data UserResponse
	Success bool
	Code    int16
}

func GetUserFromId() fiber.Handler{
	return func(c fiber.Ctx) error{
		user_id , err := FetchUserId(c)
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(GetUserResponse{
				Success: false,
				Code:400,
			})
		}
		fmt.Println("user id in get user function: ", user_id)
		u_id, err := strconv.Atoi(user_id)
		if err != nil {
			// Handle error if user_id is not a valid number
			fmt.Println("Conversion error:", err)
			return c.Status(fiber.StatusBadRequest).JSON(GetUserResponse{
				Success: false,
				Code:400,
			})
		}
		var user models.User
		err = connect.Db.NewSelect().Model((&user)).Where("id = ?",u_id).Scan(c.Context())
		if err!=nil{
			return c.Status(fiber.StatusBadRequest).JSON(GetUserResponse{
				Success: false,
				Code:400,
			})
		}

		resp_user := UserResponse{
			ID: user.ID,
			Name:user.Name,
			Email:user.Email,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		}

		return c.Status(fiber.StatusAccepted).JSON(GetUserResponse{
			Data:resp_user,
			Success: true,
			Code: 200,
		})

	}
}
