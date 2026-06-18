package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/rob121/kanban/internal/auth"
	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
	"gorm.io/gorm"
)

type AccountHandler struct {
	Render *Renderer
}

func (h *AccountHandler) Settings(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUser(r)
	if !ok {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodGet {
		_ = h.Render.Render(w, "account/settings.html", buildPage(w, r, "Account settings", user, map[string]any{
			"Saved": r.URL.Query().Get("saved") == "1",
		}))
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var editUser models.User
	if err := database.DB.First(&editUser, user.ID).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	editUser.Name = strings.TrimSpace(r.FormValue("name"))

	if editUser.Provider != "google" {
		editUser.Email = strings.TrimSpace(strings.ToLower(r.FormValue("email")))
		editUser.AvatarURL = strings.TrimSpace(r.FormValue("avatar_url"))

		username := strings.TrimSpace(r.FormValue("username"))
		if username != "" {
			editUser.Username = &username
		} else {
			editUser.Username = nil
		}

		if pw := r.FormValue("password"); pw != "" {
			hash, err := auth.HashPassword(pw)
			if err != nil {
				http.Error(w, "could not hash password", http.StatusInternalServerError)
				return
			}
			editUser.PasswordHash = hash
		}
	}

	if editUser.Name == "" {
		_ = h.Render.Render(w, "account/settings.html", buildPageError(w, r, "Account settings", &editUser, map[string]any{
			"Saved": false,
		}, "Display name is required"))
		return
	}

	if editUser.Provider != "google" && editUser.Email == "" {
		_ = h.Render.Render(w, "account/settings.html", buildPageError(w, r, "Account settings", &editUser, map[string]any{
			"Saved": false,
		}, "Email is required"))
		return
	}

	var conflict models.User
	conflictQuery := database.DB.Where("id <> ?", editUser.ID)
	if editUser.Username != nil && *editUser.Username != "" {
		conflictQuery = conflictQuery.Where("email = ? OR username = ?", editUser.Email, *editUser.Username)
	} else {
		conflictQuery = conflictQuery.Where("email = ?", editUser.Email)
	}
	if err := conflictQuery.First(&conflict).Error; err == nil {
		_ = h.Render.Render(w, "account/settings.html", buildPageError(w, r, "Account settings", &editUser, map[string]any{
			"Saved": false,
		}, "Email or username already in use"))
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	if err := database.DB.Save(&editUser).Error; err != nil {
		http.Error(w, "could not save account", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/account/settings?saved=1", http.StatusSeeOther)
}

func (h *AccountHandler) UpdateTheme(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUser(r)
	if !ok {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	theme := normalizeTheme(r.FormValue("theme"))
	database.DB.Model(user).Update("theme", theme)

	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func normalizeTheme(theme string) string {
	if strings.ToLower(strings.TrimSpace(theme)) == "dark" {
		return "dark"
	}
	return "light"
}

func userTheme(user any) string {
	if u, ok := user.(*models.User); ok && u != nil {
		return normalizeTheme(u.Theme)
	}
	return ""
}

func wantsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}
