package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
	"gorm.io/gorm"
)

var ErrInvalidAPIToken = errors.New("invalid api token")

type contextKey string

const apiUserContextKey contextKey = "apiUser"

const apiTokenPrefix = "kbn_"

func SetAPIUser(r *http.Request, user *models.User) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), apiUserContextKey, user))
}

func GetAPIUser(r *http.Request) (*models.User, bool) {
	user, ok := r.Context().Value(apiUserContextKey).(*models.User)
	return user, ok
}

func hashAPIToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func tokenPrefix(token string) string {
	if len(token) <= 12 {
		return token
	}
	return token[:12]
}

// CreateAPIToken issues a new bearer token for a user. The raw token is returned once.
func CreateAPIToken(userID uint, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "default"
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := apiTokenPrefix + hex.EncodeToString(raw)

	record := models.APIToken{
		UserID:    userID,
		Name:      name,
		Prefix:    tokenPrefix(token),
		TokenHash: hashAPIToken(token),
	}
	if err := database.DB.Create(&record).Error; err != nil {
		return "", err
	}
	return token, nil
}

// ReplaceAPIToken removes existing tokens for the user and issues a new one.
func ReplaceAPIToken(userID uint, name string) (string, error) {
	if err := database.DB.Where("user_id = ?", userID).Delete(&models.APIToken{}).Error; err != nil {
		return "", err
	}
	return CreateAPIToken(userID, name)
}

func AuthenticateAPIToken(header string) (*models.User, error) {
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" || !strings.HasPrefix(token, apiTokenPrefix) {
		return nil, ErrInvalidAPIToken
	}

	prefix := tokenPrefix(token)
	var record models.APIToken
	err := database.DB.Where("prefix = ?", prefix).First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidAPIToken
		}
		return nil, err
	}
	if record.TokenHash != hashAPIToken(token) {
		return nil, ErrInvalidAPIToken
	}

	var user models.User
	if err := database.DB.First(&user, record.UserID).Error; err != nil {
		return nil, ErrInvalidAPIToken
	}
	if user.Archived || !user.IsAPIUser() {
		return nil, ErrInvalidAPIToken
	}

	now := time.Now()
	_ = database.DB.Model(&record).Update("last_used_at", now).Error

	return &user, nil
}
