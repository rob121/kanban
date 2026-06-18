package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/rob121/kanban/internal/auth"
	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
	"gorm.io/gorm"
)

type AdminHandler struct {
	Render *Renderer
}

func (h *AdminHandler) UsersIndex(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	var users []models.User
	database.DB.Order("name asc, email asc").Find(&users)

	_ = h.Render.Render(w, "admin/users.html", buildPage(w, r, "Users", user, map[string]any{
		"Users": users,
	}))
}

func (h *AdminHandler) UserCreate(w http.ResponseWriter, r *http.Request) {
	admin, _ := auth.GetUser(r)

	if r.Method == http.MethodGet {
		_ = h.Render.Render(w, "admin/user_new.html", buildPage(w, r, "Invite User", admin, map[string]any{}))
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	newUser := models.User{
		Name:      strings.TrimSpace(r.FormValue("name")),
		Email:     strings.TrimSpace(strings.ToLower(r.FormValue("email"))),
		IsAdmin:   r.FormValue("is_admin") == "on",
		Provider:  "local",
	}
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	formData := map[string]any{
		"Name":     newUser.Name,
		"Email":    newUser.Email,
		"Username": username,
		"IsAdmin":  newUser.IsAdmin,
	}

	if newUser.Name == "" || newUser.Email == "" || username == "" {
		_ = h.Render.Render(w, "admin/user_new.html", buildPageError(w, r, "Invite User", admin, formData, "Name, username, and email are required"))
		return
	}
	if password == "" {
		_ = h.Render.Render(w, "admin/user_new.html", buildPageError(w, r, "Invite User", admin, formData, "A temporary password is required"))
		return
	}

	var conflict models.User
	if err := database.DB.Where("email = ? OR username = ?", newUser.Email, username).First(&conflict).Error; err == nil {
		_ = h.Render.Render(w, "admin/user_new.html", buildPageError(w, r, "Invite User", admin, formData, "Email or username already in use"))
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		http.Error(w, "could not hash password", http.StatusInternalServerError)
		return
	}

	newUser.Username = &username
	newUser.PasswordHash = hash

	if err := database.DB.Create(&newUser).Error; err != nil {
		http.Error(w, "could not create user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (h *AdminHandler) UserEdit(w http.ResponseWriter, r *http.Request) {
	admin, _ := auth.GetUser(r)
	userID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var editUser models.User
	if err := database.DB.First(&editUser, userID).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodGet {
		_ = h.Render.Render(w, "admin/user_edit.html", buildPage(w, r, "Edit User", admin, map[string]any{
			"EditUser": editUser,
		}))
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	editUser.Name = strings.TrimSpace(r.FormValue("name"))
	editUser.Email = strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	editUser.IsAdmin = r.FormValue("is_admin") == "on"

	if editUser.Provider != "google" {
		editUser.AvatarURL = strings.TrimSpace(r.FormValue("avatar_url"))
	}

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

	if editUser.Name == "" || editUser.Email == "" {
		_ = h.Render.Render(w, "admin/user_edit.html", buildPageError(w, r, "Edit User", admin, map[string]any{
			"EditUser": editUser,
		}, "Name and email are required"))
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
		_ = h.Render.Render(w, "admin/user_edit.html", buildPageError(w, r, "Edit User", admin, map[string]any{
			"EditUser": editUser,
		}, "Email or username already in use"))
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	if err := database.DB.Save(&editUser).Error; err != nil {
		http.Error(w, "could not save user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func parseMemberPermissions(r *http.Request) (bool, bool, bool, bool, bool) {
	return r.FormValue("can_create") == "on",
		r.FormValue("can_update") == "on",
		r.FormValue("can_delete") == "on",
		r.FormValue("can_move") == "on",
		r.FormValue("can_attach") == "on"
}

func parseUserIDForm(r *http.Request) (uint, error) {
	id, err := strconv.ParseUint(r.FormValue("user_id"), 10, 64)
	return uint(id), err
}
