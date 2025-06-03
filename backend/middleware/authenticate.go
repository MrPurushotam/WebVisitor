package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/MrPurushotam/web-visitor/utils"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		token := ""

		if authHeader != "" {
			// Check if it follows Bearer format (Bearer {token})
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			} else {
				// If no Bearer prefix, use as-is (backward compatibility)
				token = authHeader
			}
		}

		// If no token in header, try cookie
		if token == "" {
			cookie, err := c.Cookie("session_token")
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "Unauthorized",
					"message": "No session token provided",
					"success": false,
				})
				c.Abort()
				return
			}
			token = cookie
		}

		userID, err := utils.ValidateSession(token)
		fmt.Printf("%s, userID=%d, err=%v", token, userID, err)
		if err != nil {
			log.Printf("Invalid token error ", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Invalid or expired session",
				"success": false,
			})
			c.Abort()
			return
		}

		// Store user ID in context for use in handlers
		c.Set("userId", userID)
		c.Next()
	}
}
