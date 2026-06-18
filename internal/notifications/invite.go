package notifications

import (
	"fmt"
	"strings"

	"github.com/rob121/kanban/internal/config"
	"github.com/rob121/kanban/internal/models"
	"github.com/rob121/kanban/internal/users"
	"github.com/rob121/kanban/mailer"
)

// SendUserInvite emails sign-in details to a newly created user.
func SendUserInvite(user models.User, username, tempPassword string) error {
	if !mailer.Enabled() {
		return fmt.Errorf("email is not configured")
	}
	email := strings.TrimSpace(user.Email)
	if email == "" {
		return fmt.Errorf("user has no email address")
	}

	site := config.C.Branding.AppName
	if site == "" {
		site = "Kanban"
	}

	body, err := renderNotificationBody("user_invite.html", map[string]any{
		"RecipientName": user.Name,
		"SiteName":      site,
		"LoginURL":      users.LoginURL(),
		"Username":      username,
		"Password":      tempPassword,
		"BrandColor":    config.C.Branding.BrandColor,
	})
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("%s — Your account is ready", site)
	preheader := fmt.Sprintf("Sign in to %s with your new account", site)

	return mailer.New().
		To(email).
		Add("Preheader", preheader).
		Subject(subject).
		Body(body).
		Send()
}
