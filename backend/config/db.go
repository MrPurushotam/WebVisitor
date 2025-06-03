package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func Connect() error {
	uri := os.Getenv("MYSQL_URI")
	if uri == "" {
		return fmt.Errorf("MYSQL_URI environment variable is not set")
	}

	var err error
	// Open connection with retry mechanism
	for attempts := 1; attempts <= 3; attempts++ {
		log.Printf("Connecting to database (attempt %d/3)...", attempts)

		DB, err = sql.Open("mysql", uri)
		if err != nil {
			log.Printf("Error opening database connection: %v", err)
			time.Sleep(time.Second * 2)
			continue
		}

		// Configure connection pool
		DB.SetMaxOpenConns(25)
		DB.SetMaxIdleConns(5)
		DB.SetConnMaxLifetime(5 * time.Minute)

		// Test connection
		if err = DB.Ping(); err == nil {
			log.Println("Database connection established successfully.")
			return nil
		}

		log.Printf("Error pinging the database: %v", err)
		time.Sleep(time.Second * 2)
	}

	return fmt.Errorf("failed to connect to database after 3 attempts: %v", err)
}

func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}
