package routes

import (
	"log"
	"net/http"
	"strings"

	db "github.com/MrPurushotam/web-visitor/config"
	"github.com/MrPurushotam/web-visitor/middleware"
	"github.com/MrPurushotam/web-visitor/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
)

var validate *validator.Validate

type CreateUserRequest struct {
	Name     string `json:"name" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func getUserDetails(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User not authenticated",
			"success": false,
		})
		return
	}

	var user User
	query := "SELECT id, name, email FROM users WHERE id = ?"
	err := db.DB.QueryRow(query, userID).Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		log.Printf("Error fetching user details: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "User not found",
			"message": "Unable to fetch user details",
			"success": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user": gin.H{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		},
	})

}

func createUser(c *gin.Context) {
	var req CreateUserRequest

	if err := c.ShouldBindBodyWithJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"message": err.Error(),
			"success": false,
		})
		return
	}

	// Validate request body
	if err := validate.Struct(req); err != nil {
		var validationErrors []string
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Tag() {
			case "required":
				validationErrors = append(validationErrors, err.Field()+" is required")
			case "min":
				validationErrors = append(validationErrors, err.Field()+" is too short")
			case "max":
				validationErrors = append(validationErrors, err.Field()+" is too long")
			case "email":
				validationErrors = append(validationErrors, "Invalid email format")
			}
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": strings.Join(validationErrors, ", "),
			"success": false,
		})
		return
	}

	// check if user already exists
	var count int
	query := "SELECT COUNT(*) FROM users WHERE email=?"
	err := db.DB.QueryRow(query, req.Email).Scan(&count)
	if err != nil {
		log.Printf("Error checking user existence: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Error checking user existence",
			"success": false,
		})
		return
	}

	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "User exists",
			"message": "User already exists",
			"success": false,
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Error processing password",
			"success": false,
		})
		return
	}

	query = "INSERT INTO users(name,email,password) VALUES(?,?,?)"
	_, err = db.DB.Exec(query, req.Name, req.Email, string(hashedPassword))
	if err != nil {
		log.Printf("Error creating user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Error creating user",
			"success": false,
		})
		return
	}
	newUser := gin.H{"name": req.Name, "email": req.Email}
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "User created",
		"user":    newUser,
	})
}

func userLogin(c *gin.Context) {
	var req LoginUserRequest

	if err := c.ShouldBindBodyWithJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"message": err.Error(),
			"success": false,
		})
		return
	}

	// Validate request body
	if err := validate.Struct(req); err != nil {
		var validationErrors []string
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Tag() {
			case "required":
				validationErrors = append(validationErrors, err.Field()+" is required")
			case "email":
				validationErrors = append(validationErrors, "Invalid email format")
			}
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": strings.Join(validationErrors, ", "),
			"success": false,
		})
		return
	}

	var user User
	query := "SELECT id,name,email,password FROM users WHERE email = ? "

	err := db.DB.QueryRow(query, req.Email).Scan(&user.ID, &user.Name, &user.Email, &user.Password)
	if err != nil {
		log.Printf("User doesn't exist: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Email doesn't exist",
			"success": false,
			"message": "User doesn't exist",
		})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		log.Printf("Invalid password: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Invalid credentials",
			"success": false,
			"message": "Email or password is incorrect",
		})
		return
	}

	// Create session
	token, err := utils.CreateSession(user.ID)
	if err != nil {
		log.Printf("Error creating session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "Failed to create session",
			"success": false,
		})
		return
	}

	// Set session cookie
	c.SetCookie("session_token", token, 60*60*1000, "/", "", false, true)

	userData := gin.H{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Login successful",
		"user":    userData,
	})
}

func logoutUser(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		cookie, err := c.Cookie("session_token")
		if err == nil {
			token = cookie
		}
	}

	if token != "" {
		// This should handle removing the token from auth_token table
		if err := utils.InvalidateSession(token); err != nil {
			log.Printf("Error invalidating session: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal server error",
				"message": "Error logging out",
				"success": false,
			})
			return
		}
	}

	// Clear cookie
	c.SetCookie("session_token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out successfully",
	})

}

func verifyUser(c *gin.Context) {
	// Implement user verification logic here
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Verification endpoint",
	})
}

func resendVerification(c *gin.Context) {
	email := c.Param("email")
	// Implement resend verification logic here
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Verification email sent to " + email,
	})
}

func InitUserRouter(rg *gin.RouterGroup) {
	validate = validator.New()

	router := rg.Group("/user")

	// Public routes
	router.POST("/create/", createUser)
	router.POST("/login/", userLogin)
	router.POST("/verify/", verifyUser)
	router.POST("/resend/:email", resendVerification)

	// protected routes
	protected := router.Group("/") // Creates a sub-group within "/user"
	protected.Use(middleware.AuthMiddleware()) // Applies auth middleware only to this sub-group
	{
		protected.GET("/", getUserDetails)      
		protected.POST("/logout/", logoutUser)
	}
}
