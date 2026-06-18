package server

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/rob121/kanban/internal/handlers"
	"github.com/rob121/kanban/internal/middleware"
	apiv1 "github.com/rob121/kanban/internal/api/v1"
)

type Server struct {
	mux     *http.ServeMux
	render  *handlers.Renderer
	auth    *handlers.AuthHandler
	boards  *handlers.BoardHandler
	cards   *handlers.CardHandler
	cats    *handlers.CategoryHandler
	members *handlers.MemberHandler
	admin   *handlers.AdminHandler
	tags    *handlers.TagHandler
	attach  *handlers.AttachmentHandler
	account *handlers.AccountHandler
	static  embed.FS
}

func New(views, static embed.FS) (*Server, error) {
	render, err := handlers.NewRenderer(views)
	if err != nil {
		return nil, err
	}

	s := &Server{
		mux:     http.NewServeMux(),
		render:  render,
		auth:    &handlers.AuthHandler{Render: render},
		boards:  &handlers.BoardHandler{Render: render},
		cards:   &handlers.CardHandler{Render: render},
		cats:    &handlers.CategoryHandler{Render: render},
		members: &handlers.MemberHandler{Render: render},
		admin:   &handlers.AdminHandler{Render: render},
		tags:    &handlers.TagHandler{Render: render},
		attach:  &handlers.AttachmentHandler{Render: render},
		account: &handlers.AccountHandler{Render: render},
		static:  static,
	}
	s.routes()
	return s, nil
}

func (s *Server) Handler() http.Handler {
	return middleware.EnsureCSRF(middleware.SkipCSRFForAPI(s.mux))
}

func (s *Server) routes() {
	staticFS, _ := fs.Sub(s.static, "static")
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	s.mux.Handle("GET /login", middleware.RedirectIfAuth(http.HandlerFunc(s.auth.Login)))
	s.mux.Handle("POST /login", middleware.RedirectIfAuth(http.HandlerFunc(s.auth.Login)))
	s.mux.Handle("GET /auth/google", http.HandlerFunc(s.auth.GoogleBegin))
	s.mux.Handle("GET /auth/google/callback", http.HandlerFunc(s.auth.GoogleCallback))
	s.mux.Handle("POST /logout", middleware.RequireAuth(http.HandlerFunc(s.auth.Logout)))
	s.mux.Handle("GET /account/settings", middleware.RequireAuth(http.HandlerFunc(s.account.Settings)))
	s.mux.Handle("POST /account/settings", middleware.RequireAuth(http.HandlerFunc(s.account.Settings)))
	s.mux.Handle("POST /account/theme", middleware.RequireAuth(http.HandlerFunc(s.account.UpdateTheme)))

	s.mux.Handle("GET /admin/users", middleware.RequireAuth(middleware.RequireAdmin(http.HandlerFunc(s.admin.UsersIndex))))
	s.mux.Handle("GET /admin/users/new", middleware.RequireAuth(middleware.RequireAdmin(http.HandlerFunc(s.admin.UserCreate))))
	s.mux.Handle("POST /admin/users", middleware.RequireAuth(middleware.RequireAdmin(http.HandlerFunc(s.admin.UserCreate))))
	s.mux.Handle("GET /admin/users/{id}", middleware.RequireAuth(middleware.RequireAdmin(http.HandlerFunc(s.admin.UserEdit))))
	s.mux.Handle("GET /admin/users/{id}/token", middleware.RequireAuth(middleware.RequireAdmin(http.HandlerFunc(s.admin.UserShowToken))))
	s.mux.Handle("POST /admin/users/{id}", middleware.RequireAuth(middleware.RequireAdmin(http.HandlerFunc(s.admin.UserEdit))))
	s.mux.Handle("POST /admin/users/{id}/archive", middleware.RequireAuth(middleware.RequireAdmin(http.HandlerFunc(s.admin.UserArchive))))
	s.mux.Handle("POST /admin/users/{id}/delete", middleware.RequireAuth(middleware.RequireAdmin(http.HandlerFunc(s.admin.UserDelete))))
	s.mux.Handle("POST /admin/users/{id}/regenerate-token", middleware.RequireAuth(middleware.RequireAdmin(http.HandlerFunc(s.admin.UserRegenerateToken))))

	s.mux.Handle("GET /boards", middleware.RequireAuth(http.HandlerFunc(s.boards.Index)))
	s.mux.Handle("GET /boards/new", middleware.RequireAuth(http.HandlerFunc(s.boards.Create)))
	s.mux.Handle("POST /boards", middleware.RequireAuth(http.HandlerFunc(s.boards.Create)))
	s.mux.Handle("GET /boards/{id}", middleware.RequireAuth(http.HandlerFunc(s.boards.Show)))
	s.mux.Handle("GET /boards/{id}/settings", middleware.RequireAuth(http.HandlerFunc(s.boards.Settings)))
	s.mux.Handle("POST /boards/{id}/settings", middleware.RequireAuth(http.HandlerFunc(s.boards.Settings)))
	s.mux.Handle("POST /boards/{id}/archive", middleware.RequireAuth(http.HandlerFunc(s.boards.Archive)))
	s.mux.Handle("POST /boards/{id}/delete", middleware.RequireAuth(middleware.RequireAdmin(http.HandlerFunc(s.boards.Delete))))
	s.mux.Handle("GET /boards/{id}/members", middleware.RequireAuth(http.HandlerFunc(s.members.Manage)))
	s.mux.Handle("POST /boards/{id}/members", middleware.RequireAuth(http.HandlerFunc(s.members.Add)))
	s.mux.Handle("POST /boards/{id}/members/{userId}", middleware.RequireAuth(http.HandlerFunc(s.members.Update)))
	s.mux.Handle("POST /boards/{id}/members/{userId}/remove", middleware.RequireAuth(http.HandlerFunc(s.members.Remove)))

	s.mux.Handle("GET /boards/{id}/tags", middleware.RequireAuth(http.HandlerFunc(s.tags.Manage)))
	s.mux.Handle("POST /boards/{id}/tags", middleware.RequireAuth(http.HandlerFunc(s.tags.Create)))
	s.mux.Handle("POST /boards/{id}/tags/{tagId}/delete", middleware.RequireAuth(http.HandlerFunc(s.tags.Delete)))

	s.mux.Handle("POST /categories", middleware.RequireAuth(http.HandlerFunc(s.cats.Create)))
	s.mux.Handle("POST /categories/{id}/move", middleware.RequireAuth(http.HandlerFunc(s.cats.Move)))
	s.mux.Handle("POST /categories/{id}/delete", middleware.RequireAuth(http.HandlerFunc(s.cats.Delete)))

	s.mux.Handle("POST /cards", middleware.RequireAuth(http.HandlerFunc(s.cards.Create)))
	s.mux.Handle("GET /cards/{id}", middleware.RequireAuth(http.HandlerFunc(s.cards.Show)))
	s.mux.Handle("POST /cards/{id}", middleware.RequireAuth(http.HandlerFunc(s.cards.Update)))
	s.mux.Handle("POST /cards/{id}/move", middleware.RequireAuth(http.HandlerFunc(s.cards.Move)))
	s.mux.Handle("POST /cards/{id}/archive", middleware.RequireAuth(http.HandlerFunc(s.cards.Archive)))
	s.mux.Handle("POST /cards/{id}/comments", middleware.RequireAuth(http.HandlerFunc(s.cards.AddComment)))
	s.mux.Handle("POST /cards/{id}/subscribe", middleware.RequireAuth(http.HandlerFunc(s.cards.Subscribe)))
	s.mux.Handle("POST /cards/{id}/unsubscribe", middleware.RequireAuth(http.HandlerFunc(s.cards.Unsubscribe)))
	s.mux.Handle("POST /cards/{id}/attachments", middleware.RequireAuth(http.HandlerFunc(s.attach.Upload)))
	s.mux.Handle("GET /attachments/{id}", middleware.RequireAuth(http.HandlerFunc(s.attach.Serve)))
	s.mux.Handle("POST /attachments/{id}/delete", middleware.RequireAuth(http.HandlerFunc(s.attach.Delete)))

	apiv1.Register(s.mux)

	s.mux.Handle("GET /", middleware.RedirectIfAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})))
}
