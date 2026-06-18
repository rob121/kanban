package v1

import (
	"net/http"

	"github.com/rob121/kanban/internal/auth"
	"github.com/rob121/kanban/internal/models"
	"github.com/rob121/kanban/internal/permissions"
)

func currentUser(r *http.Request) (*models.User, bool) {
	return auth.GetAPIUser(r)
}

func requireBoardAccess(w http.ResponseWriter, r *http.Request, user *models.User, boardID uint, check func(permissions.Access) bool) (permissions.Access, bool) {
	access, err := permissions.GetAccess(user, boardID)
	if err != nil || !check(access) {
		writeError(w, http.StatusNotFound, "not found")
		return permissions.Access{}, false
	}
	return access, true
}

func userJSON(u models.User) map[string]any {
	out := map[string]any{
		"id":        u.ID,
		"name":      u.Name,
		"email":     u.Email,
		"user_type": u.UserType,
		"is_admin":  u.IsAdmin,
	}
	if u.Username != nil {
		out["username"] = *u.Username
	}
	return out
}
