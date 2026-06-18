package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rob121/kanban/internal/auth"
)

func SkipCSRFForAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		CSRF(next).ServeHTTP(w, r)
	})
}

func RequireAPIToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := auth.AuthenticateAPIToken(r.Header.Get("Authorization"))
		if err != nil {
			writeAPIError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, auth.SetAPIUser(r, user))
	})
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
