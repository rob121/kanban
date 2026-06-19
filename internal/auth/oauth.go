package auth

import (
	"strings"

	"github.com/markbates/goth"
)

func oauthDisplayName(u goth.User) string {
	if n := strings.TrimSpace(u.Name); n != "" {
		return n
	}
	first := strings.TrimSpace(u.FirstName)
	last := strings.TrimSpace(u.LastName)
	if first != "" || last != "" {
		return strings.TrimSpace(first + " " + last)
	}
	if n := strings.TrimSpace(u.NickName); n != "" {
		return n
	}
	return ""
}
