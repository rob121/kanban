package auth

import (
	"net/http"
)

const apiTokenFlashKey = "api_token_flash"
const apiTokenRegeneratedKey = "api_token_regenerated"

// SetAPITokenFlash stores a one-time API token in the session for display after redirect.
func SetAPITokenFlash(w http.ResponseWriter, r *http.Request, token string, regenerated bool) error {
	sess, err := store.Get(r, sessionName)
	if err != nil {
		return err
	}
	sess.Values[apiTokenFlashKey] = token
	sess.Values[apiTokenRegeneratedKey] = regenerated
	return sess.Save(r, w)
}

// ConsumeAPITokenFlash returns and clears a flashed API token, if present.
func ConsumeAPITokenFlash(w http.ResponseWriter, r *http.Request) (token string, regenerated bool, ok bool) {
	sess, err := store.Get(r, sessionName)
	if err != nil {
		return "", false, false
	}
	raw, exists := sess.Values[apiTokenFlashKey].(string)
	if !exists || raw == "" {
		return "", false, false
	}
	regenerated, _ = sess.Values[apiTokenRegeneratedKey].(bool)
	delete(sess.Values, apiTokenFlashKey)
	delete(sess.Values, apiTokenRegeneratedKey)
	_ = sess.Save(r, w)
	return raw, regenerated, true
}
