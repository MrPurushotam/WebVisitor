package routes

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	db "github.com/MrPurushotam/web-visitor/config"
	"github.com/MrPurushotam/web-visitor/middleware"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AddUriRequest struct {
	Url  string `json:"url" validate:"required,min=5,max=500"`
	Name string `json:"name" validate:"required,min=3,max=100"`
}
type EditUriRequest struct {
	Url  string `json:"url" validate:"omitempty,min=5,max=500"`
	Name string `json:"name" validate:"omitempty,min=3,max=100"`
}

// normalizeURL validates and normalizes the URL
func normalizeURL(rawURL string) (string, error) {
	// Add http:// if no scheme is provided
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	// Parse and validate URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %v", err)
	}

	// Validate scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("URL must use HTTP or HTTPS protocol")
	}

	// Validate host
	if parsedURL.Host == "" {
		return "", fmt.Errorf("URL must have a valid host")
	}

	// Check for suspicious or blocked domains (optional security measure)
	if isBlockedDomain(parsedURL.Host) {
		return "", fmt.Errorf("domain is not allowed")
	}

	// Return normalized URL
	return parsedURL.String(), nil
}

// testURLAccessibility tests if the URL is accessible
func testURLAccessibility(testURL string) (status string, responseTime int, responseCode int, errorMessage string) {
	client := &http.Client{
		Timeout: 15 * time.Second, // Increased timeout for better reliability
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	// Create HEAD request first (lightweight)
	request, err := http.NewRequest("HEAD", testURL, nil)
	if err != nil {
		return "error", 0, 0, fmt.Sprintf("Failed to create request: %v", err)
	}

	// Set realistic browser headers
	setHTTPHeaders(request)

	start := time.Now()
	resp, err := client.Do(request)
	responseTime = int(time.Since(start).Milliseconds())

	if err != nil {
		// Try GET request if HEAD fails (some servers don't support HEAD)
		request, err = http.NewRequest("GET", testURL, nil)
		if err != nil {
			return "error", responseTime, 0, fmt.Sprintf("Failed to create GET request: %v", err)
		}

		setHTTPHeaders(request)

		start = time.Now()
		resp, err = client.Do(request)
		responseTime = int(time.Since(start).Milliseconds())

		if err != nil {
			return "error", responseTime, 0, err.Error()
		}
	}

	defer resp.Body.Close()
	responseCode = resp.StatusCode

	// Determine status based on response code
	switch {
	case responseCode >= 200 && responseCode < 300:
		return "online", responseTime, responseCode, ""
	case responseCode >= 300 && responseCode < 400:
		return "online", responseTime, responseCode, "" // Redirects are OK
	case responseCode >= 400 && responseCode < 500:
		return "offline", responseTime, responseCode, fmt.Sprintf("Client error: %s", resp.Status)
	case responseCode >= 500:
		return "offline", responseTime, responseCode, fmt.Sprintf("Server error: %s", resp.Status)
	default:
		return "error", responseTime, responseCode, fmt.Sprintf("Unexpected response: %s", resp.Status)
	}
}

// setHTTPHeaders sets realistic browser headers
func setHTTPHeaders(request *http.Request) {
	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	request.Header.Set("Accept-Language", "en-US,en;q=0.9")
	request.Header.Set("Accept-Encoding", "gzip, deflate, br")
	request.Header.Set("Connection", "keep-alive")
	request.Header.Set("Upgrade-Insecure-Requests", "1")
	request.Header.Set("Sec-Fetch-Dest", "document")
	request.Header.Set("Sec-Fetch-Mode", "navigate")
	request.Header.Set("Sec-Fetch-Site", "none")
	request.Header.Set("DNT", "1")
}

// isBlockedDomain checks if a domain should be blocked (optional security measure)
func isBlockedDomain(host string) bool {
	// Remove port if present
	if colonIndex := strings.LastIndex(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

	// List of blocked domains/patterns
	blockedDomains := []string{
		"localhost",
		"127.0.0.1",
		"0.0.0.0",
		"::1",
	}

	host = strings.ToLower(host)
	for _, blocked := range blockedDomains {
		if host == blocked || strings.HasSuffix(host, "."+blocked) {
			return true
		}
	}

	// Block private IP ranges (optional - you might want to allow these in development)
	if isPrivateIP(host) {
		return true
	}

	return false
}

// isPrivateIP checks if the host is a private IP address
func isPrivateIP(host string) bool {
	// Basic check for private IP ranges
	// 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
	return strings.HasPrefix(host, "10.") ||
		strings.HasPrefix(host, "192.168.") ||
		(strings.HasPrefix(host, "172.") &&
			len(strings.Split(host, ".")) >= 2 &&
			strings.Split(host, ".")[1] >= "16" &&
			strings.Split(host, ".")[1] <= "31")
}

func addUri(c *gin.Context) {
	var req AddUriRequest

	if err := c.ShouldBindJSON(&req); err != nil {
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
				validationErrors = append(validationErrors, fmt.Sprintf("%s is required", err.Field()))
			case "min":
				validationErrors = append(validationErrors, fmt.Sprintf("%s is too short (minimum %s characters)", err.Field(), err.Param()))
			case "max":
				validationErrors = append(validationErrors, fmt.Sprintf("%s is too long (maximum %s characters)", err.Field(), err.Param()))
			case "url":
				validationErrors = append(validationErrors, "Invalid URL format")
			default:
				validationErrors = append(validationErrors, fmt.Sprintf("%s is invalid", err.Field()))
			}
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": strings.Join(validationErrors, ", "),
			"success": false,
		})
		return
	}

	// Get user ID from context (set by AuthMiddleware)
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User not authenticated",
			"success": false,
		})
		return
	}

	// Parse and validate URL before making a request
	parsedURL, err := url.Parse(req.Url)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid URL format",
			"message": "URL must be a valid HTTP or HTTPS URL",
			"success": false,
		})
		return
	}

	// Normalize and validate URL
	normalizedURL, err := normalizeURL(req.Url)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid URL format",
			"message": err.Error(),
			"success": false,
		})
		return
	}

	// Check if URL already exists for this user
	var existingID int
	err = db.DB.QueryRow(
		"SELECT id FROM urls WHERE user_id = ? AND url = ?",
		userID, normalizedURL,
	).Scan(&existingID)

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "URL already exists",
			"message": "This URL is already being monitored",
			"success": false,
		})
		return
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Failed to check URL existence",
			"success": false,
		})
		return
	}

	// Test URL accessibility
	urlStatus, responseTime, responseCode, errorMessage := testURLAccessibility(normalizedURL)

	// Insert URL into database
	result, err := db.DB.Exec(
		"INSERT INTO urls (user_id, url, name, status, response_time, last_checked) VALUES (?, ?, ?, ?, ?, NOW())",
		userID, normalizedURL, req.Name, urlStatus, responseTime,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Failed to save URL",
			"success": false,
		})
		return
	}

	urlID, _ := result.LastInsertId()

	// Log the first check
	_, err = db.DB.Exec(
		"INSERT INTO logs (url_id, status, response_time, response_code, error_message, checked_at) VALUES (?, ?, ?, ?, ?, NOW())",
		urlID, urlStatus, responseTime, responseCode, errorMessage,
	)

	if err != nil {
		// Log error but don't fail the request since URL was created successfully
		fmt.Printf("Warning: Failed to log initial URL check: %v\n", err)
	}

	// Return success response
	c.JSON(http.StatusCreated, gin.H{
		"message": "URL added successfully",
		"success": true,
		"data": gin.H{
			"id":            urlID,
			"url":           normalizedURL,
			"name":          req.Name,
			"status":        urlStatus,
			"response_time": responseTime,
			"response_code": responseCode,
		},
	})
}

func editUri(c *gin.Context) {
	uriID := c.Param("id")

	if uriID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing parameter",
			"message": "URI ID is required",
			"success": false,
		})
		return
	}

	var req EditUriRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"message": err.Error(),
			"success": false,
		})
		return
	}

	// Check if at least one field is provided
	if req.Url == "" && req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": "At least one field (URL or name) must be provided",
			"success": false,
		})
		return
	}

	// Validate request body
	if err := validate.Struct(req); err != nil {
		var validationErrors []string
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Tag() {
			case "min":
				validationErrors = append(validationErrors, fmt.Sprintf("%s is too short (minimum %s characters)", err.Field(), err.Param()))
			case "max":
				validationErrors = append(validationErrors, fmt.Sprintf("%s is too long (maximum %s characters)", err.Field(), err.Param()))
			case "url":
				validationErrors = append(validationErrors, "Invalid URL format")
			default:
				validationErrors = append(validationErrors, fmt.Sprintf("%s is invalid", err.Field()))
			}
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": strings.Join(validationErrors, ", "),
			"success": false,
		})
		return
	}

	// Get user ID from context (set by AuthMiddleware)
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User not authenticated",
			"success": false,
		})
		return
	}
	var existingURL, existingName, existingStatus string
	var existingResponseTime, existingResponseCode int

	err := db.DB.QueryRow(
		"SELECT url, name, status, response_time, 0 FROM urls WHERE id = ? AND user_id = ?",
		uriID, userID,
	).Scan(&existingURL, &existingName, &existingStatus, &existingResponseTime, &existingResponseCode)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "URI not found",
				"message": "The URI doesn't exist or doesn't belong to you",
				"success": false,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Database error",
				"message": "Failed to retrieve URI information",
				"success": false,
			})
		}
		return
	}
	// Variables to store the values we'll update
	normalizedURL := existingURL
	newName := existingName
	status := existingStatus
	responseTime := existingResponseTime
	responseCode := existingResponseCode
	var errorMessage string

	// If URL is being updated, validate and normalize it
	if req.Url != "" && req.Url != existingURL {
		var err error
		normalizedURL, err = normalizeURL(req.Url)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid URL format",
				"message": err.Error(),
				"success": false,
			})
			return
		}

		// Check if the new URL already exists for this user
		var duplicateID int
		err = db.DB.QueryRow(
			"SELECT id FROM urls WHERE user_id = ? AND url = ? AND id != ?",
			userID, normalizedURL, uriID,
		).Scan(&duplicateID)

		if err == nil {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "URL already exists",
				"message": "This URL is already being monitored",
				"success": false,
			})
			return
		} else if err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Database error",
				"message": "Failed to check URL existence",
				"success": false,
			})
			return
		}

		// Test the new URL's accessibility
		status, responseTime, responseCode, errorMessage = testURLAccessibility(normalizedURL)
	}

	if req.Name != "" {
		newName = req.Name
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Failed to start transaction",
			"success": false,
		})
		return
	}
	var result sql.Result
	if req.Url != "" && req.Url != existingURL {
		result, err = tx.Exec(
			"UPDATE urls SET url = ?, name = ?, status = ?, response_time = ?, last_checked = NOW() WHERE id = ? AND user_id = ?",
			normalizedURL, newName, status, responseTime, uriID, userID,
		)
	} else {
		result, err = tx.Exec(
			"UPDATE urls SET name=? WHERE id=? AND user_id=?", newName, uriID, userID,
		)
	}

	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to update url",
			"error":   "Database error",
		})
		return
	}
	rowsAffected, _ := result.RowsAffected()

	if rowsAffected == 0 {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "URI not found",
			"message": "The URI doesn't exist or doesn't belong to you",
			"success": false,
		})
		return
	}

	if req.Url != "" && req.Url != existingURL {
		_, err := tx.Exec(
			"INSERT INTO logs (url_id, status, response_time, response_code, error_message, checked_at) VALUES (?, ?, ?, ?, ?, NOW())",
			uriID, status, responseTime, responseCode, errorMessage,
		)

		if err != nil {
			// Log error but don't fail the request
			fmt.Printf("Warning: Failed to log URL check after edit: %v\n", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Failed to commit changes",
			"success": false,
		})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"message": "URL updated successfully",
		"success": true,
		"data": gin.H{
			"id":            uriID,
			"url":           normalizedURL,
			"name":          newName,
			"status":        status,
			"response_time": responseTime,
			"response_code": responseCode,
		},
	})
}

func getAllUri(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User not authenticated",
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

	// Query to get total count
	var totalCount int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM urls WHERE user_id = ?", userID).Scan(&totalCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Failed to get URL count",
			"success": false,
		})
		return
	}

	// Query to get URLs with pagination
	rows, err := db.DB.Query(`
        SELECT id, url, name, status, response_time, last_checked, created_at 
        FROM urls 
        WHERE user_id = ? 
        ORDER BY created_at DESC 
        LIMIT ? OFFSET ?
    `, userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Failed to retrieve URLs",
			"success": false,
		})
		return
	}
	defer rows.Close()

	urls := []gin.H{}

	for rows.Next() {
		var (
			id           int64
			url          string
			name         string
			status       string
			responseTime int
			lastChecked  time.Time
			createdAt    time.Time
		)

		if err := rows.Scan(&id, &url, &name, &status, &responseTime, &lastChecked, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Database error",
				"message": "Failed to parse URL data",
				"success": false,
			})
			return
		}
		// Get the latest log entry for this URL
		var (
			latestStatus    string
			latestResTime   int
			latestResCode   int
			latestCheckedAt time.Time
		)

		// Query to get the latest log entry
		logErr := db.DB.QueryRow(`
            SELECT status, response_time, response_code, checked_at 
            FROM logs 
            WHERE url_id = ? 
            ORDER BY checked_at DESC 
            LIMIT 1
        `, id).Scan(&latestStatus, &latestResTime, &latestResCode, &latestCheckedAt)

		// Add URL data to the slice
		urlData := gin.H{
			"id":            id,
			"url":           url,
			"name":          name,
			"status":        status,
			"response_time": responseTime,
			"last_checked":  lastChecked.Format(time.RFC3339),
			"created_at":    createdAt.Format(time.RFC3339),
		}

		// Add latest log data if available
		if logErr == nil {
			urlData["latest_check"] = gin.H{
				"status":        latestStatus,
				"response_time": latestResTime,
				"response_code": latestResCode,
				"checked_at":    latestCheckedAt.Format(time.RFC3339),
			}
		}

		urls = append(urls, urlData)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Error iterating through URLs",
			"success": false,
		})
		return
	}

	// Return response with pagination info
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "URLs retrieved successfully",
		"data": gin.H{
			"urls": urls,
			"pagination": gin.H{
				"total":  totalCount,
				"limit":  limit,
				"offset": offset,
				"pages":  (totalCount + limit - 1) / limit,
			},
		},
	})
}

func deleteUri(c *gin.Context) {
	// Get URI ID from route parameter
	uriID := c.Param("id")

	if uriID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing parameter",
			"message": "URI ID is required",
			"success": false,
		})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User not authenticated",
			"success": false,
		})
		return
	}

	// Start a transaction
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Failed to start transaction",
			"success": false,
		})
		return
	}

	// First, verify the URL exists and belongs to the user
	var urlName string
	var logCount int

	err = tx.QueryRow(
		"SELECT name FROM urls WHERE id = ? AND user_id = ?",
		uriID, userID,
	).Scan(&urlName)

	if err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "URI not found",
				"message": "The URI doesn't exist or doesn't belong to you",
				"success": false,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Database error",
				"message": "Failed to retrieve URI information",
				"success": false,
			})
		}
		return
	}

	// Count how many logs will be deleted
	err = tx.QueryRow("SELECT COUNT(*) FROM logs WHERE url_id = ?", uriID).Scan(&logCount)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Failed to count logs",
			"success": false,
		})
		return
	}

	// Delete the URI (logs will be deleted via ON DELETE CASCADE)
	result, err := tx.Exec("DELETE FROM urls WHERE id = ? AND user_id = ?", uriID, userID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Failed to delete URL",
			"success": false,
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "URI not found",
			"message": "The URI doesn't exist or doesn't belong to you",
			"success": false,
		})
		return
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Failed to commit changes",
			"success": false,
		})
		return
	}

	// Return success response with information about deleted items
	c.JSON(http.StatusOK, gin.H{
		"message": "URL and all associated logs deleted successfully",
		"success": true,
		"data": gin.H{
			"url_id":     uriID,
			"url_name":   urlName,
			"logs_count": logCount,
		},
	})
}

func InitUriRouter(rg *gin.RouterGroup) {
	router := rg.Group("/uri")
	router.Use(middleware.AuthMiddleware())

	{
		router.POST("/", addUri)
		router.GET("/", getAllUri)
		router.PUT("/:id", editUri)
		router.DELETE("/:id", deleteUri)
	}
}
