package utils

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"log"
	"time"

	db "github.com/MrPurushotam/web-visitor/config"
)

func GenerateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func CreateSession(userId int) (string, error) {
	token, err := GenerateSessionToken()
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().Add(1 * time.Hour)
	query := "INSERT INTO auth_tokens(user_id,token,expires_at) VALUES (?,?,?)"

	_, err = db.DB.Exec(query, userId, token, expiresAt)
	if err != nil {
		return "", err
	}
	return token, nil
}

func ValidateSession(token string) (int, error) {
	var userID int
	var expiresAt sql.NullTime

	query := `SELECT user_id, expires_at FROM auth_tokens WHERE token=? AND is_active=TRUE AND expires_at >= NOW()`

	err := db.DB.QueryRow(query, token).Scan(&userID, &expiresAt)
	if err != nil {
		log.Printf("%s%v", token, err)
		return 0, err
	}

	updateQuery := `UPDATE auth_tokens SET last_used_at = NOW() WHERE token=?`
	_, updateErr := db.DB.Exec(updateQuery, token)
	if updateErr != nil {
		log.Printf("Failed to update last_used_at: %v", updateErr)
	}

	return userID, nil
}

func InvalidateSession(token string) error {
	query := `UPDATE auth_tokens SET is_active=FALSE WHERE token=?`
	_, err := db.DB.Exec(query, token)
	return err
}
