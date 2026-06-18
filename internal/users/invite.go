package users

import (
	"fmt"
	"strings"

	"github.com/rob121/kanban/internal/config"
)

func LoginURL() string {
	return strings.TrimRight(config.C.BaseURL, "/") + "/login"
}

func InviteText(name, username, password string) string {
	app := config.C.Branding.AppName
	if app == "" {
		app = "Kanban"
	}
	return fmt.Sprintf(`Hi %s,

You've been invited to %s. Sign in with:

URL: %s
Username: %s
Password: %s

Please change your password after signing in.`, name, app, LoginURL(), username, password)
}
