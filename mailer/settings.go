package mailer

import (
	"strings"
)

// Settings holds SMTP and template options loaded from application config.
type Settings struct {
	Host         string
	Port         int
	User         string
	Password     string
	From         string
	FromName     string
	ReplyTo      string
	TestTo       string
	TemplatePath string
	Logo         string
	BaseURL      string
	SiteName     string
}

var configuredBaseURL string
var configuredSiteName string

// Configure applies mail settings from application config. Call once after config.Load().
func Configure(s Settings) {
	Host = strings.TrimSpace(s.Host)
	Port = s.Port
	if Port == 0 {
		Port = 587
	}
	UserName = strings.TrimSpace(s.User)
	Password = s.Password
	From = strings.TrimSpace(s.From)
	FromName = strings.TrimSpace(s.FromName)
	ReplyTo = strings.TrimSpace(s.ReplyTo)
	TestTo = strings.TrimSpace(s.TestTo)
	TestMode = TestTo != ""
	TmplPath = strings.TrimSpace(s.TemplatePath)
	Logo = strings.TrimSpace(s.Logo)
	configuredBaseURL = strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	configuredSiteName = strings.TrimSpace(s.SiteName)
}

// Enabled reports whether outbound SMTP is configured (host is set).
func Enabled() bool {
	return Host != ""
}

func defaultMailerBaseURL() string {
	return configuredBaseURL
}

func defaultMailerSiteName() string {
	if configuredSiteName != "" {
		return configuredSiteName
	}
	return "Kanban"
}
