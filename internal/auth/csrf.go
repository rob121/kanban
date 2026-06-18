package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"mime"
	"net/http"
	"strings"
)

const csrfSessionKey = "csrf_token"

func EnsureCSRF(w http.ResponseWriter, r *http.Request) string {
	sess, err := store.Get(r, sessionName)
	if err != nil {
		return ""
	}
	if token, ok := sess.Values[csrfSessionKey].(string); ok && token != "" {
		return token
	}
	token, err := newCSRFToken()
	if err != nil {
		return ""
	}
	sess.Values[csrfSessionKey] = token
	_ = sess.Save(r, w)
	return token
}

func CSRFToken(r *http.Request) string {
	sess, err := store.Get(r, sessionName)
	if err != nil {
		return ""
	}
	token, _ := sess.Values[csrfSessionKey].(string)
	return token
}

func ValidateCSRF(r *http.Request) bool {
	_ = ParseRequestForm(r)
	expected := CSRFToken(r)
	if expected == "" {
		return false
	}
	provided := r.FormValue("csrf_token")
	if provided == "" {
		provided = r.Header.Get("X-CSRF-Token")
	}
	if provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

// ParseRequestForm parses POST bodies for both urlencoded and multipart forms.
func ParseRequestForm(r *http.Request) error {
	if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
		return nil
	}
	if r.MultipartForm != nil {
		return nil
	}
	if r.PostForm != nil {
		return nil
	}

	ct := r.Header.Get("Content-Type")
	if ct == "" {
		return r.ParseForm()
	}
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return r.ParseForm()
	}
	if strings.EqualFold(mediaType, "multipart/form-data") {
		return r.ParseMultipartForm(32 << 20)
	}
	return r.ParseForm()
}

func newCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
