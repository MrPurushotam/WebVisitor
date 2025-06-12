package schema

import (
	"log"
	"strings"

	db "github.com/MrPurushotam/web-visitor/config"
)

func CreateSchema() error {
	userSchema := `
		CREATE TABLE IF NOT EXISTS users(
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(150) NOT NULL UNIQUE,
			password VARCHAR(255) NOT NULL,
			verified BOOLEAN DEFAULT FALSE,
			tier ENUM('free','premium') DEFAULT 'free',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		);`

	urlSchema := `
		CREATE TABLE IF NOT EXISTS urls(
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id INT NOT NULL,
			url VARCHAR(500) NOT NULL,
			name VARCHAR(100),
			` + "`interval`" + ` ENUM('6hr','12hr') DEFAULT '6hr',
			custom_interval INT DEFAULT NULL,

			last_checked TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			status ENUM('online', 'offline', 'error') DEFAULT 'online',
			response_time INT DEFAULT 0,
			
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			INDEX idx_user_active (user_id, status),
			INDEX idx_last_checked (last_checked)
		);`

	logsSchema := `
		CREATE TABLE IF NOT EXISTS logs(
			id INT AUTO_INCREMENT PRIMARY KEY,
			url_id INT NOT NULL,
			status ENUM('online', 'offline', 'error') NOT NULL,
			response_time INT DEFAULT 0,
			response_code INT DEFAULT 0,
			error_message TEXT,
			checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (url_id) REFERENCES urls(id) ON DELETE CASCADE,
			INDEX idx_url_status (url_id, checked_at)
		);`

	authSchema := `
		CREATE TABLE IF NOT EXISTS auth_tokens(
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id INT NOT NULL,
			token VARCHAR(255) NOT NULL UNIQUE,
			expires_at TIMESTAMP NULL,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_used_at TIMESTAMP NULL,
			user_agent VARCHAR(255) NULL,
			ip_address VARCHAR(45) NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			INDEX idx_token (token),
			INDEX idx_user_active (user_id, is_active)
		);
	`
	schemas := []string{userSchema, urlSchema, logsSchema, authSchema}

	for _, schema := range schemas {
		if _, err := db.DB.Exec((schema)); err != nil {
			log.Printf("Error creating schema: %v", err)
			return err
		}
		log.Println("Database schema created successfully")
	}
	return nil
}

func CreateIndex() error {
	indexes := []string{
		`CREATE INDEX idx_websites_next_check ON urls(last_checked, ` + "`interval`" + `);`,
		`CREATE INDEX idx_logs_recent ON logs(checked_at DESC);`,
		`CREATE INDEX idx_users_email ON users(email);`,
	}
	for _, index := range indexes {
		_, err := db.DB.Exec(index)
		if err != nil {
			if isDuplicateKeyError(err) {
				log.Printf("Index already exists: %s", index)
				continue
			}
			log.Printf("Error creating index: %v", err)
			return err
		}
		log.Println("Index created successfully: ", index)
	}
	return nil
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	duplicateKeyPhrases := []string{
		"Error 1061",
		"Duplicate key name",
		"index already exists",
		"Index already exists",
		"relation already exists",
	}

	for _, phrase := range duplicateKeyPhrases {
		if strings.Contains(errMsg, phrase) {
			return true
		}
	}

	return false
}
