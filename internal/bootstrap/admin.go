package bootstrap

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/rob121/kanban/internal/auth"
	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
	"gorm.io/gorm"
)

type AdminOptions struct {
	Username string
	Email    string
	Password string
	Name     string
	Update   bool
}

func CreateAdmin(opts AdminOptions) (string, error) {
	opts.Username = strings.TrimSpace(opts.Username)
	opts.Email = strings.TrimSpace(strings.ToLower(opts.Email))
	opts.Name = strings.TrimSpace(opts.Name)

	if opts.Username == "" {
		return "", errors.New("username is required")
	}
	if opts.Email == "" {
		return "", errors.New("email is required")
	}
	if opts.Name == "" {
		opts.Name = opts.Username
	}

	generated := false
	if opts.Password == "" {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return "", fmt.Errorf("generate password: %w", err)
		}
		opts.Password = hex.EncodeToString(b)
		generated = true
	}

	hash, err := auth.HashPassword(opts.Password)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	username := opts.Username

	var byUsername models.User
	err = database.DB.Where("username = ?", username).First(&byUsername).Error
	if err == nil {
		if !opts.Update {
			return "", fmt.Errorf("user %q already exists (email %q); re-run with --update to replace credentials", username, byUsername.Email)
		}
		return finishAdmin(&byUsername, opts, hash, generated, true)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	var byEmail models.User
	err = database.DB.Where("email = ?", opts.Email).First(&byEmail).Error
	if err == nil {
		if opts.Update && byEmail.UsernameString() == "" {
			byEmail.Username = &username
			return finishAdmin(&byEmail, opts, hash, generated, true)
		}
		return "", fmt.Errorf("email %q is already used by %q; choose a different email or use --update with that account's username", opts.Email, byEmail.UsernameString())
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	user := models.User{
		Username:     &username,
		Email:        opts.Email,
		Name:         opts.Name,
		PasswordHash: hash,
		Provider:     "local",
		IsAdmin:      true,
	}
	if err := database.DB.Create(&user).Error; err != nil {
		return "", fmt.Errorf("create admin: %w", err)
	}

	if generated {
		return opts.Password, nil
	}
	return "", nil
}

func finishAdmin(user *models.User, opts AdminOptions, hash string, generated, updated bool) (string, error) {
	user.Email = opts.Email
	user.Name = opts.Name
	user.PasswordHash = hash
	user.Provider = "local"
	user.IsAdmin = true
	if user.Username == nil || *user.Username != opts.Username {
		u := opts.Username
		user.Username = &u
	}
	if err := database.DB.Save(user).Error; err != nil {
		if updated {
			return "", fmt.Errorf("update admin: %w", err)
		}
		return "", fmt.Errorf("create admin: %w", err)
	}
	if generated {
		return opts.Password, nil
	}
	return "", nil
}
