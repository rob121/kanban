package config

import (
	"strconv"
	"strings"

	"github.com/rob121/vhelp"
	"github.com/spf13/viper"
)

type DatabaseConfig struct {
	Driver string
	DSN    string
}

type GoogleAuthConfig struct {
	ClientID     string
	ClientSecret string
}

type AuthConfig struct {
	Google GoogleAuthConfig
}

type BrandingConfig struct {
	AppName    string
	BrandMark  string
	BrandColor string
}

type MailConfig struct {
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
}

type AppConfig struct {
	Port           int
	BaseURL        string
	SessionSecret  string
	AttachmentsDir string
	Database       DatabaseConfig
	Auth           AuthConfig
	Branding       BrandingConfig
	Mail           MailConfig
}

var C AppConfig

func Load() {
	vhelp.Load("config")
	v, err := vhelp.Get("config")
	if err != nil {
		panic(err)
	}
	C = AppConfig{
		Port:           v.GetInt("port"),
		BaseURL:        v.GetString("base_url"),
		SessionSecret:  v.GetString("session_secret"),
		AttachmentsDir: v.GetString("attachments_dir"),
		Database: DatabaseConfig{
			Driver: v.GetString("database.driver"),
			DSN:    v.GetString("database.dsn"),
		},
		Auth: AuthConfig{
			Google: GoogleAuthConfig{
				ClientID:     v.GetString("auth.google.client_id"),
				ClientSecret: v.GetString("auth.google.client_secret"),
			},
		},
		Branding: BrandingConfig{
			AppName:    v.GetString("branding.app_name"),
			BrandMark:  v.GetString("branding.brand_mark"),
			BrandColor: normalizeHexColor(v.GetString("branding.brand_color"), "#4f46e5"),
		},
		Mail: MailConfig{
			Host:         v.GetString("mail.host"),
			Port:         v.GetInt("mail.port"),
			User:         v.GetString("mail.user"),
			Password:     v.GetString("mail.password"),
			From:         v.GetString("mail.from"),
			FromName:     v.GetString("mail.from_name"),
			ReplyTo:      v.GetString("mail.reply_to"),
			TestTo:       v.GetString("mail.test_to"),
			TemplatePath: v.GetString("mail.template_path"),
			Logo:         v.GetString("mail.logo"),
		},
	}
	if C.Port == 0 {
		C.Port = 8080
	}
	if C.BaseURL == "" {
		C.BaseURL = "http://localhost:8080"
	}
	if C.Branding.AppName == "" {
		C.Branding.AppName = "Kanban"
	}
	if C.Branding.BrandMark == "" {
		C.Branding.BrandMark = "K"
	}
}

func Viper() *viper.Viper {
	v, _ := vhelp.Get("config")
	return v
}

func normalizeHexColor(color, fallback string) string {
	color = strings.TrimSpace(color)
	if len(color) == 7 && color[0] == '#' {
		if _, err := strconv.ParseUint(color[1:], 16, 24); err == nil {
			return strings.ToLower(color)
		}
	}
	return fallback
}
