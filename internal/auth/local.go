package auth

import (
	"errors"

	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
	"gorm.io/gorm"
)

var ErrInvalidCredentials = errors.New("invalid username or password")

func AuthenticateLocal(username, password string) (*models.User, error) {
	var user models.User
	err := database.DB.Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if user.PasswordHash == "" || !CheckPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}
	if user.Archived {
		return nil, ErrInvalidCredentials
	}
	if user.IsAPIUser() {
		return nil, ErrInvalidCredentials
	}
	return &user, nil
}
