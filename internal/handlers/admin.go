package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/rob121/kanban/internal/auth"
	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
	"github.com/rob121/kanban/internal/notifications"
	"github.com/rob121/kanban/internal/users"
	"github.com/rob121/kanban/mailer"
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
		_ = h.Render.Render(w, "admin/user_new.html", buildPage(w, r, "Invite User", admin, map[string]any{
			"MailEnabled": mailer.Enabled(),
			"UserType":    models.UserTypeWeb,
		}))
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	userType := strings.TrimSpace(r.FormValue("user_type"))
	if userType == "" {
		userType = models.UserTypeWeb
	}

	newUser := models.User{
		Name:     strings.TrimSpace(r.FormValue("name")),
		Email:    strings.TrimSpace(strings.ToLower(r.FormValue("email"))),
		IsAdmin:  r.FormValue("is_admin") == "on",
		UserType: userType,
	}
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	formData := map[string]any{
		"Name":        newUser.Name,
		"Email":       newUser.Email,
		"Username":    username,
		"IsAdmin":     newUser.IsAdmin,
		"UserType":    userType,
		"MailEnabled": mailer.Enabled(),
		"InviteNow":   r.FormValue("invite_now") == "on",
	}

	if newUser.Name == "" || newUser.Email == "" {
		_ = h.Render.Render(w, "admin/user_new.html", buildPageError(w, r, "Invite User", admin, formData, "Name and email are required"))
		return
	}

	if userType == models.UserTypeAPI {
		h.createAPIUser(w, r, admin, newUser, username, formData)
		return
	}

	newUser.Provider = "local"
	if username == "" {
		_ = h.Render.Render(w, "admin/user_new.html", buildPageError(w, r, "Invite User", admin, formData, "Username is required for web users"))
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

	inviteNow := r.FormValue("invite_now") == "on"
	emailSent := false
	emailError := ""
	if inviteNow {
		if err := notifications.SendUserInvite(newUser, username, password); err != nil {
			emailError = err.Error()
		} else {
			emailSent = true
		}
	}

	_ = h.Render.Render(w, "admin/user_created.html", buildPage(w, r, "User Created", admin, map[string]any{
		"NewUser":    newUser,
		"Username":   username,
		"Password":   password,
		"LoginURL":   users.LoginURL(),
		"InviteText": users.InviteText(newUser.Name, username, password),
		"InvitedNow": inviteNow,
		"EmailSent":  emailSent,
		"EmailError": emailError,
	}))
}

func (h *AdminHandler) createAPIUser(w http.ResponseWriter, r *http.Request, admin *models.User, newUser models.User, username string, formData map[string]any) {
	conflictQuery := database.DB.Where("email = ?", newUser.Email)
	if username != "" {
		conflictQuery = database.DB.Where("email = ? OR username = ?", newUser.Email, username)
	}
	var conflict models.User
	if err := conflictQuery.First(&conflict).Error; err == nil {
		_ = h.Render.Render(w, "admin/user_new.html", buildPageError(w, r, "Invite User", admin, formData, "Email or username already in use"))
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	newUser.Provider = "api"
	if username != "" {
		newUser.Username = &username
	}

	if err := database.DB.Create(&newUser).Error; err != nil {
		http.Error(w, "could not create user", http.StatusInternalServerError)
		return
	}

	token, err := auth.CreateAPIToken(newUser.ID, "default")
	if err != nil {
		http.Error(w, "could not create api token", http.StatusInternalServerError)
		return
	}

	if err := auth.SetAPITokenFlash(w, r, token, false); err != nil {
		http.Error(w, "could not save token", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/users/"+strconv.FormatUint(uint64(newUser.ID), 10)+"/token", http.StatusSeeOther)
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
		assigned, _ := users.AssignedCardCount(editUser.ID)
		canDelete, deleteReason := users.CanHardDelete(editUser.ID)
		var apiToken models.APIToken
		hasToken := editUser.IsAPIUser() && database.DB.Where("user_id = ?", editUser.ID).Order("created_at desc").First(&apiToken).Error == nil
		_ = h.Render.Render(w, "admin/user_edit.html", buildPage(w, r, "Edit User", admin, map[string]any{
			"EditUser":       editUser,
			"AssignedCards":  assigned,
			"CanDeleteUser":  canDelete,
			"DeleteReason":   deleteReason,
			"IsSelf":         admin.ID == editUser.ID,
			"HasAPIToken":    hasToken,
			"APITokenPrefix": apiToken.Prefix,
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

	if editUser.IsAPIUser() {
		if editUser.Name == "" || editUser.Email == "" {
			_ = h.Render.Render(w, "admin/user_edit.html", buildPageError(w, r, "Edit User", admin, map[string]any{
				"EditUser": editUser,
			}, "Name and email are required"))
			return
		}
		var conflict models.User
		if err := database.DB.Where("id <> ? AND email = ?", editUser.ID, editUser.Email).First(&conflict).Error; err == nil {
			_ = h.Render.Render(w, "admin/user_edit.html", buildPageError(w, r, "Edit User", admin, map[string]any{
				"EditUser": editUser,
			}, "Email already in use"))
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
		return
	}

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

func (h *AdminHandler) UserRegenerateToken(w http.ResponseWriter, r *http.Request) {
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
	if !editUser.IsAPIUser() {
		http.NotFound(w, r)
		return
	}

	token, err := auth.ReplaceAPIToken(editUser.ID, "default")
	if err != nil {
		http.Error(w, "could not regenerate token", http.StatusInternalServerError)
		return
	}

	if err := auth.SetAPITokenFlash(w, r, token, true); err != nil {
		http.Error(w, "could not save token", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/users/"+strconv.FormatUint(uint64(userID), 10)+"/token", http.StatusSeeOther)
}

func (h *AdminHandler) UserShowToken(w http.ResponseWriter, r *http.Request) {
	admin, _ := auth.GetUser(r)
	userID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	token, regenerated, ok := auth.ConsumeAPITokenFlash(w, r)
	if !ok {
		http.Redirect(w, r, "/admin/users/"+strconv.FormatUint(uint64(userID), 10), http.StatusSeeOther)
		return
	}

	var editUser models.User
	if err := database.DB.First(&editUser, userID).Error; err != nil || !editUser.IsAPIUser() {
		http.NotFound(w, r)
		return
	}

	_ = h.Render.Render(w, "admin/api_user_created.html", buildPage(w, r, "API Token", admin, map[string]any{
		"NewUser":     editUser,
		"APIToken":    token,
		"DocsURL":     "/api/v1/docs/",
		"Regenerated": regenerated,
	}))
}

func (h *AdminHandler) UserArchive(w http.ResponseWriter, r *http.Request) {
	admin, _ := auth.GetUser(r)
	userID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := users.Archive(admin.ID, userID); err != nil {
		var editUser models.User
		if database.DB.First(&editUser, userID).Error != nil {
			http.NotFound(w, r)
			return
		}
		assigned, _ := users.AssignedCardCount(editUser.ID)
		canDelete, deleteReason := users.CanHardDelete(editUser.ID)
		_ = h.Render.Render(w, "admin/user_edit.html", buildPageError(w, r, "Edit User", admin, map[string]any{
			"EditUser":      editUser,
			"AssignedCards": assigned,
			"CanDeleteUser": canDelete,
			"DeleteReason":  deleteReason,
			"IsSelf":        admin.ID == editUser.ID,
		}, archiveErrorMessage(err)))
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (h *AdminHandler) UserDelete(w http.ResponseWriter, r *http.Request) {
	admin, _ := auth.GetUser(r)
	userID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := users.HardDelete(admin.ID, userID); err != nil {
		var editUser models.User
		if database.DB.First(&editUser, userID).Error != nil {
			http.NotFound(w, r)
			return
		}
		assigned, _ := users.AssignedCardCount(editUser.ID)
		canDelete, deleteReason := users.CanHardDelete(editUser.ID)
		_ = h.Render.Render(w, "admin/user_edit.html", buildPageError(w, r, "Edit User", admin, map[string]any{
			"EditUser":      editUser,
			"AssignedCards": assigned,
			"CanDeleteUser": canDelete,
			"DeleteReason":  deleteReason,
			"IsSelf":        admin.ID == editUser.ID,
		}, err.Error()))
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func archiveErrorMessage(err error) string {
	switch {
	case errors.Is(err, users.ErrSelfAction):
		return "You cannot archive your own account"
	case errors.Is(err, users.ErrActiveAdmin):
		return "Cannot archive the only active administrator"
	case errors.Is(err, users.ErrAlreadyArchived):
		return "This user is already archived"
	default:
		return "Could not archive user"
	}
}

func parseMemberPermissions(r *http.Request) (bool, bool, bool, bool, bool, bool) {
	return r.FormValue("can_create") == "on",
		r.FormValue("can_update") == "on",
		r.FormValue("can_delete") == "on",
		r.FormValue("can_move") == "on",
		r.FormValue("can_attach") == "on",
		r.FormValue("can_manage_tags") == "on"
}

func parseUserIDForm(r *http.Request) (uint, error) {
	id, err := strconv.ParseUint(r.FormValue("user_id"), 10, 64)
	return uint(id), err
}
