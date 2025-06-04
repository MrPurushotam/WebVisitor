package main

import (
	"log"
	"net/http"
	"os"

	db "github.com/MrPurushotam/web-visitor/config"
	schema "github.com/MrPurushotam/web-visitor/libs"
	"github.com/MrPurushotam/web-visitor/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	mode := os.Getenv("GIN_MODE")
	switch mode {
	case "release":
		gin.SetMode(gin.ReleaseMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	r := gin.Default()
	setupSwaggerRoutes(r)

	log.Println("Connecting to database...")
	if err := db.Connect(); err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.CloseDB()

	log.Println("Creating database schema...")
	if err := schema.CreateSchema(); err != nil {
		log.Fatalf("Database schema creation failed: %v", err)
	}

	log.Println("Creating database indexes...")
	if err := schema.CreateIndex(); err != nil {
		log.Fatalf("Database index creation failed: %v", err)
	}

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "API is running."})
	})


	routes.Init(r)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	} 	
}
