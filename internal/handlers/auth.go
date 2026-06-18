package handlers

import (
	"errors"
	"net/http"

	"github.com/markbates/goth/gothic"
	"github.com/rob121/kanban/internal/auth"
)

type AuthHandler struct {
	Render *Renderer
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.loginPost(w, r)
		return
	}

	msg := ""
	switch r.URL.Query().Get("error") {
	case "auth_failed":
		msg = "Authentication failed. Please try again."
	case "no_account":
		msg = "No account exists for that email. Contact an administrator to be invited."
	case "session":
		msg = "Could not start a session."
	case "invalid_credentials":
		msg = "Invalid username or password."
	}

	_ = h.Render.Render(w, "auth/login.html", buildPageError(w, r, "Sign in", nil, nil, msg))
}

func (h *AuthHandler) loginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	user, err := auth.AuthenticateLocal(r.FormValue("username"), r.FormValue("password"))
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			http.Redirect(w, r, "/login?error=invalid_credentials", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/login?error=auth_failed", http.StatusSeeOther)
		return
	}

	if err := auth.SetUser(w, r, user); err != nil {
		http.Redirect(w, r, "/login?error=session", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/boards", http.StatusSeeOther)
}

func (h *AuthHandler) GoogleBegin(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	q.Set("provider", "google")
	r.URL.RawQuery = q.Encode()
	gothic.BeginAuthHandler(w, r)
}

func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	q.Set("provider", "google")
	r.URL.RawQuery = q.Encode()

	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		http.Redirect(w, r, "/login?error=auth_failed", http.StatusSeeOther)
		return
	}

	user, err := auth.FindOAuthUser(gothUser)
	if err != nil {
		if errors.Is(err, auth.ErrOAuthUserNotFound) {
			http.Redirect(w, r, "/login?error=no_account", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/login?error=auth_failed", http.StatusSeeOther)
		return
	}

	if err := auth.SetUser(w, r, user); err != nil {
		http.Redirect(w, r, "/login?error=session", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/boards", http.StatusSeeOther)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	auth.ClearUser(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
