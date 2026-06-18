package handlers

import (
	"net/http"

	"github.com/rob121/kanban/internal/auth"
	"github.com/rob121/kanban/internal/config"
)

func buildPage(w http.ResponseWriter, r *http.Request, title string, user any, data any) PageData {
	return PageData{
		Title:      title,
		User:       user,
		UserTheme:  userTheme(user),
		Data:       data,
		CSRFToken:  auth.EnsureCSRF(w, r),
		BrandName:  config.C.Branding.AppName,
		BrandMark:  config.C.Branding.BrandMark,
		BrandColor: config.C.Branding.BrandColor,
	}
}

func buildPageError(w http.ResponseWriter, r *http.Request, title string, user any, data any, errMsg string) PageData {
	pd := buildPage(w, r, title, user, data)
	pd.Error = errMsg
	return pd
}
