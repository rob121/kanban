package v1

import (
	"net/http"
)

type Handler struct{}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, userJSON(*user))
}
