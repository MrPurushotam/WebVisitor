package routes

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	db "github.com/MrPurushotam/web-visitor/config"
	"github.com/MrPurushotam/web-visitor/middleware"
	"github.com/gin-gonic/gin"
)

func getAllLogs(c *gin.Context) {
	urlId := c.Param("id")
	fmt.Print(urlId)
	if urlId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing parameter",
			"message": "URL ID is required",
			"success": false,
		})
		return
	}

	// Get user ID from context for security check
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User not authenticated",
			"success": false,
		})
		return
	}

	// Verify the URL belongs to the authenticated user
	var urlExists bool
	err := db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM urls WHERE id = ?)", urlId).Scan(&urlExists)
	if err != nil {
		fmt.Printf("Database error checking URL existence: %v\n", err)
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Access denied",
			"message": "Failed to verify URL existence",
			"success": false,
		})
		return
	}

	if !urlExists {
		fmt.Printf("URL with ID %s doesn't exist\n", urlId)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "URL not found",
			"message": "The specified URL doesn't exist",
			"success": false,
		})
		return
	}

	var urlBelongsToUser bool
	err = db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM urls WHERE id = ? AND user_id = ?)", urlId, userID).Scan(&urlBelongsToUser)
	if err != nil {
		fmt.Printf("Database error checking URL ownership: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Failed to verify URL ownership",
			"success": false,
		})
		return
	}

	// If URL exists but doesn't belong to the user
	if !urlBelongsToUser {
		fmt.Printf("URL %s exists but doesn't belong to user %v\n", urlId, userID)
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Access denied",
			"message": "You don't have access to logs for this URL",
			"success": false,
		})
		return
	}

	// Handle pagination
	limit := 10 // Default limit
	offset := 0 // Default offset

	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if pageParam := c.Query("page"); pageParam != "" {
		if parsedPage, err := strconv.Atoi(pageParam); err == nil && parsedPage > 0 {
			offset = (parsedPage - 1) * limit
		}
	}

	var totalCount int
	err = db.DB.QueryRow("SELECT COUNT(*) FROM logs WHERE url_id=?", urlId).Scan(&totalCount)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Failed to get total log count",
			"success": false,
		})
		return
	}

	// Query to get URLs with pagination
    rows, err := db.DB.Query(`
        SELECT id, url_id, status, response_time, response_code, error_message, checked_at 
        FROM logs 
        WHERE url_id = ? 
        ORDER BY checked_at DESC 
        LIMIT ? OFFSET ?
    `, urlId, limit, offset)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Failed to retrieve logs",
			"success": false,
		})
		return
	}

	defer rows.Close()

	logs := []gin.H{}

	for rows.Next() {
		var (
			id            int64
			url_id        int64
			status        string
			response_time int
			response_code int
			error_message string
			checked_at    time.Time
		)

		if err := rows.Scan(&id, &url_id, &status, &response_time, &response_code, &error_message, &checked_at); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Database error",
				"message": "Failed to parse log data",
				"success": false,
			})
			return
		}

		// Append log data to the slice
		logData := gin.H{
			"id":            id,
			"url_id":        url_id,
			"status":        status,
			"response_time": response_time,
			"response_code": response_code,
			"error_message": error_message,
			"checked_at":    checked_at.Format(time.RFC3339),
		}

		logs = append(logs, logData)
	}
	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Error iterating through logs",
			"success": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logs retrieved successfully",
		"data": gin.H{
			"logs": logs,
			"pagination": gin.H{
				"total":  totalCount,
				"limit":  limit,
				"offset": offset,
				"pages":  (totalCount + limit - 1) / limit,
			},
		},
	})

}

func InitLogsRouter(rg *gin.RouterGroup) {
	router := rg.Group("/logs")
	router.Use(middleware.AuthMiddleware())
	{
		router.GET("/:id", getAllLogs)
	}
}
