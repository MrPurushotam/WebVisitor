package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type SwaggerSpec struct {
	data []byte
}

var swaggerSpec *SwaggerSpec

func initSwagger() error {
	swaggerPath := filepath.Join("..", "swagger.yaml")
	data, err := os.ReadFile(swaggerPath)
	if err != nil {
		// Try current directory as fallback
		data, err = os.ReadFile("swagger.yaml")
		if err != nil {
			return err
		}
	}
	swaggerSpec = &SwaggerSpec{data: data}
	return nil
}

func serveSwaggerYAML(c *gin.Context) {
	if swaggerSpec == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Swagger specification not loaded",
		})
		return
	}

	c.Header("Content-Type", "application/yaml")
	c.Data(http.StatusOK, "application/yaml", swaggerSpec.data)
}

func serveSwaggerUI(c *gin.Context) {
	html := `
	<!DOCTYPE html>
	<html>
		<head>
			<title>Web Visitor API Documentation</title>
			<link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@3.25.0/swagger-ui.css" />
		</head>
		<body>
			<div id="swagger-ui"></div>
			<script src="https://unpkg.com/swagger-ui-dist@3.25.0/swagger-ui-bundle.js"></script>
			<script>
				SwaggerUIBundle({
					url: '/swagger.json',
					dom_id: '#swagger-ui',
					presets: [
						SwaggerUIBundle.presets.apis,
						SwaggerUIBundle.presets.standalone
					]
				});
			</script>
		</body>
	</html>`

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

// setupSwaggerRoutes adds swagger routes to the router
func setupSwaggerRoutes(r *gin.Engine) {
	// Initialize swagger specification
	if err := initSwagger(); err != nil {
		panic("Failed to load swagger specification: " + err.Error())
	}

	r.GET("/swagger.yaml", serveSwaggerYAML)
	r.GET("/docs", serveSwaggerUI)
}
