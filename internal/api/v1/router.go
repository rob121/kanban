package v1

import (
	"net/http"

	"github.com/rob121/kanban/internal/middleware"
)

func Register(mux *http.ServeMux) {
	h := &Handler{}

	mux.Handle("GET /api/v1/openapi.yaml", http.HandlerFunc(h.OpenAPI))
	mux.Handle("GET /api/v1/docs", http.HandlerFunc(h.Docs))
	mux.Handle("GET /api/v1/docs/", http.HandlerFunc(h.Docs))

	auth := middleware.RequireAPIToken
	mux.Handle("GET /api/v1/me", auth(http.HandlerFunc(h.Me)))
	mux.Handle("GET /api/v1/boards", auth(http.HandlerFunc(h.ListBoards)))
	mux.Handle("GET /api/v1/boards/{id}", auth(http.HandlerFunc(h.ShowBoard)))
	mux.Handle("GET /api/v1/boards/{id}/categories", auth(http.HandlerFunc(h.ListCategories)))
	mux.Handle("GET /api/v1/boards/{id}/cards", auth(http.HandlerFunc(h.ListCards)))
	mux.Handle("GET /api/v1/cards/{id}", auth(http.HandlerFunc(h.ShowCard)))
	mux.Handle("POST /api/v1/cards", auth(http.HandlerFunc(h.CreateCard)))
	mux.Handle("PATCH /api/v1/cards/{id}", auth(http.HandlerFunc(h.UpdateCard)))
	mux.Handle("POST /api/v1/cards/{id}/move", auth(http.HandlerFunc(h.MoveCard)))
	mux.Handle("POST /api/v1/cards/{id}/archive", auth(http.HandlerFunc(h.ArchiveCard)))
	mux.Handle("POST /api/v1/cards/{id}/comments", auth(http.HandlerFunc(h.AddComment)))
}
