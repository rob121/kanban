package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
	"github.com/rob121/kanban/internal/config"
	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
	"gorm.io/gorm"
)

var ErrOAuthUserNotFound = errors.New("oauth user not found")

const sessionName = "kanban_session"
const userSessionKey = "user_id"

var store *sessions.CookieStore

func Init() {
	secret := config.C.SessionSecret
	if secret == "" || secret == "change-me-to-a-long-random-string" {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		secret = hex.EncodeToString(b)
	}

	store = sessions.NewCookieStore([]byte(secret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 14,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	}

	gothic.Store = store

	callback := config.C.BaseURL + "/auth/google/callback"
	goth.UseProviders(
		google.New(config.C.Auth.Google.ClientID, config.C.Auth.Google.ClientSecret, callback),
	)
}

func SessionStore() *sessions.CookieStore {
	return store
}

func GetUser(r *http.Request) (*models.User, bool) {
	sess, err := store.Get(r, sessionName)
	if err != nil {
		return nil, false
	}
	var id uint
	switch v := sess.Values[userSessionKey].(type) {
	case uint:
		id = v
	case uint64:
		id = uint(v)
	case int:
		id = uint(v)
	case int64:
		id = uint(v)
	case float64:
		id = uint(v)
	default:
		return nil, false
	}
	if id == 0 {
		return nil, false
	}
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return nil, false
	}
	if user.Archived {
		return nil, false
	}
	if user.IsAPIUser() {
		return nil, false
	}
	return &user, true
}

func SetUser(w http.ResponseWriter, r *http.Request, user *models.User) error {
	sess, err := store.Get(r, sessionName)
	if err != nil {
		return err
	}
	sess.Values[userSessionKey] = user.ID
	return sess.Save(r, w)
}

func ClearUser(w http.ResponseWriter, r *http.Request) {
	sess, _ := store.Get(r, sessionName)
	delete(sess.Values, userSessionKey)
	sess.Options.MaxAge = -1
	_ = sess.Save(r, w)
	gothic.Logout(w, r)
}

func FindOAuthUser(gothUser goth.User) (*models.User, error) {
	email := strings.ToLower(strings.TrimSpace(gothUser.Email))
	var user models.User
	err := database.DB.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOAuthUserNotFound
		}
		return nil, err
	}
	if user.Archived {
		return nil, ErrOAuthUserNotFound
	}
	if user.IsAPIUser() {
		return nil, ErrOAuthUserNotFound
	}

	user.Name = gothUser.Name
	user.AvatarURL = gothUser.AvatarURL
	user.Provider = gothUser.Provider
	if err := database.DB.Save(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
