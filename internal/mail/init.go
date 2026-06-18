package mail

import (
	"github.com/rob121/kanban/internal/config"
	"github.com/rob121/kanban/mailer"
)

// Init configures the mailer package from application config.
func Init() {
	m := config.C.Mail
	mailer.Configure(mailer.Settings{
		Host:         m.Host,
		Port:         m.Port,
		User:         m.User,
		Password:     m.Password,
		From:         m.From,
		FromName:     m.FromName,
		ReplyTo:      m.ReplyTo,
		TestTo:       m.TestTo,
		TemplatePath: m.TemplatePath,
		Logo:         m.Logo,
		BaseURL:      config.C.BaseURL,
		SiteName:     config.C.Branding.AppName,
	})
}
