package handlers

import (
	"fmt"
	"math/rand"
	"time"
	"viralforge/src/connect"
	"viralforge/src/env"
	"viralforge/src/models"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/resend/resend-go/v3"
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
		
		var user models.User
		err = connect.Db.NewSelect().Model((&user)).Where("id = ?",user_id).Scan(c.Context())
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

type SendCodeResponse struct{
	Message string 
	Success bool
	

}
// for forget password :
func SendCode() fiber.Handler {
	return func(c fiber.Ctx) error {
		envs := env.NewEnv()

		var body struct {
			Email string `json:"email"`
		}

		// 1. Parse request
		if err := c.Bind().Body(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(SendCodeResponse{
				Message: "Invalid request body",
				Success: false,
			},
			)
		}

		// 2. Check user exists
		var user models.User
		err := connect.Db.NewSelect().
			Model(&user).
			Where("email = ?", body.Email).
			Scan(c.Context())

		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(SendCodeResponse{
				Message: "User with this email not found",
				Success: false,
			},)
		}

		// 3. Generate 6-digit OTP
		otp := fmt.Sprintf("%06d", rand.Intn(1000000))

		// 4. Store OTP in DB (or Redis preferred)
		// Example: add fields in User or separate table
		_, err = connect.Db.NewUpdate().
			Model(&user).
			Set("otp = ?", otp).
			Set("otp_expiry = ?", time.Now().Add(10*time.Minute)).
			Where("id = ?", user.ID).
			Exec(c.Context())

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(SendCodeResponse{
				Message: "Failed to store OTP",
				Success: false,
			})
		}

		// 5. Send email via Resend
		client := resend.NewClient(envs.RESEND_API_KEY)

		params := &resend.SendEmailRequest{
			From:    "ViralForge <noreply@viralforge.xastros.site>",
			To:      []string{body.Email},
			Subject: "Your OTP Code - ViralForge",
			Html: fmt.Sprintf(`
				<h2>Your OTP Code</h2>
				<p>Your verification code is:</p>
				<h1>%s</h1>
				<p>This code will expire in 10 minutes.</p>
			`, otp),
		}

		_, err = client.Emails.Send(params)
		if err != nil {
			fmt.Println(err.Error())
			return c.Status(fiber.StatusInternalServerError).JSON(SendCodeResponse{
				Message: "Failed to send email",
				Success: false,
			})
		}

		return c.Status(fiber.StatusOK).JSON(SendCodeResponse{
			Message: "OTP sent successfully",
			Success: true,
		})
	}
}

func VerifyOTP() fiber.Handler {
	return func(c fiber.Ctx) error {
		var body struct {
			Email string `json:"email"`
			OTP   string `json:"otp"`
		}

		if err := c.Bind().Body(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(SendCodeResponse{
				Message: "Invalid request body",
				Success: false,
			})
		}

		var user models.User
		err := connect.Db.NewSelect().
			Model(&user).
			Where("email = ?", body.Email).
			Scan(c.Context())

		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(SendCodeResponse{
				Message: "User not found",
				Success: false,
			})
		}

		// Check OTP match
		if user.OTP != body.OTP {
			return c.Status(fiber.StatusUnauthorized).JSON(SendCodeResponse{
				Message: "Invalid OTP",
				Success: false,
			})
		}

		// Check expiry
		if time.Now().After(user.OTPExpiry) {
			return c.Status(fiber.StatusUnauthorized).JSON(SendCodeResponse{
				Message: "OTP expired",
				Success: false,
			})
		}

		// Mark OTP verified (important step)
		_, err = connect.Db.NewUpdate().
			Model(&user).
			Set("is_verified = ?", true).
			Where("id = ?", user.ID).
			Exec(c.Context())

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(SendCodeResponse{
				Message: "Failed to verify OTP",
				Success: false,
			})
		}

		return c.Status(fiber.StatusOK).JSON(SendCodeResponse{
			Message: "OTP verified successfully",
			Success: true,
		})
	}
}

// new password and confirm password from the frontend , and matches these two , if true, take the new password and update the password of the email id.

func ResetPassword() fiber.Handler {
	return func(c fiber.Ctx) error {
		var body struct {
			Email           string `json:"email"`
			NewPassword     string `json:"new_password"`
			ConfirmPassword string `json:"confirm_password"`
		}

		if err := c.Bind().Body(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(SendCodeResponse{
				Message: "Invalid request body",
				Success: false,
			})
		}

		// Validate passwords match
		if body.NewPassword != body.ConfirmPassword {
			return c.Status(fiber.StatusBadRequest).JSON(SendCodeResponse{
				Message: "Passwords do not match",
				Success: false,
			})
		}

		var user models.User
		err := connect.Db.NewSelect().
			Model(&user).
			Where("email = ?", body.Email).
			Scan(c.Context())

		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(SendCodeResponse{
				Message: "User not found",
				Success: false,
			})
		}

		// Ensure OTP was verified
		if !user.IsVerified {
			return c.Status(fiber.StatusUnauthorized).JSON(SendCodeResponse{
				Message: "OTP not verified",
				Success: false,
			})
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword(
			[]byte(body.NewPassword),
			14,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(SendCodeResponse{
				Message: "Failed to hash password",
				Success: false,
			})
		}

		// Update password + clear OTP fields
		_, err = connect.Db.NewUpdate().
			Model(&user).
			Set("password = ?", string(hashedPassword)).
			Set("otp = NULL").
			Set("otp_expiry = NULL").
			Set("is_verified = ?", false).
			Where("id = ?", user.ID).
			Exec(c.Context())

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(SendCodeResponse{
				Message: "Failed to update password",
				Success: false,
			})
		}

		return c.Status(fiber.StatusOK).JSON(SendCodeResponse{
			Message: "Password reset successful",
			Success: true,
		})
	}
}

func Home() fiber.Handler{
	return func(c fiber.Ctx) error{
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Welcome to ViralForge API!",
			"success": true,
		})
	}
}
