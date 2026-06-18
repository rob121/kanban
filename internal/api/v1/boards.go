package v1

import (
	"net/http"
	"time"

	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
	"github.com/rob121/kanban/internal/permissions"
)

func (h *Handler) ListBoards(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	boards, err := permissions.AccessibleBoards(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load boards")
		return
	}

	items := make([]map[string]any, 0, len(boards))
	for _, b := range boards {
		items = append(items, boardSummaryJSON(b))
	}
	writeJSON(w, http.StatusOK, map[string]any{"boards": items})
}

func (h *Handler) ShowBoard(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	boardID, err := pathUint(r, "id")
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	access, ok := requireBoardAccess(w, r, user, boardID, permissions.Access.CanView)
	if !ok {
		return
	}

	var categories []models.Category
	database.DB.Where("board_id = ?", boardID).Order("position asc").Find(&categories)

	var tags []models.BoardTag
	database.DB.Where("board_id = ?", boardID).Order("position asc").Find(&tags)

	var cards []models.Card
	database.DB.Preload("Assignee").Preload("Creator").Preload("Tags").
		Where("board_id = ? AND archived = ?", boardID, false).
		Order("position asc").
		Find(&cards)

	cardsByCategory := make(map[uint][]map[string]any)
	for _, c := range cards {
		cardsByCategory[c.CategoryID] = append(cardsByCategory[c.CategoryID], cardJSON(c))
	}

	catItems := make([]map[string]any, 0, len(categories))
	for _, cat := range categories {
		item := categoryJSON(cat)
		item["cards"] = cardsByCategory[cat.ID]
		catItems = append(catItems, item)
	}

	tagItems := make([]map[string]any, 0, len(tags))
	for _, t := range tags {
		tagItems = append(tagItems, tagJSON(t))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"board":      boardDetailJSON(access.Board),
		"categories": catItems,
		"tags":       tagItems,
	})
}

func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	boardID, err := pathUint(r, "id")
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if _, ok := requireBoardAccess(w, r, user, boardID, permissions.Access.CanView); !ok {
		return
	}

	var categories []models.Category
	database.DB.Where("board_id = ?", boardID).Order("position asc").Find(&categories)

	items := make([]map[string]any, 0, len(categories))
	for _, cat := range categories {
		items = append(items, categoryJSON(cat))
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": items})
}

func categoryJSON(c models.Category) map[string]any {
	return map[string]any{
		"id":       c.ID,
		"board_id": c.BoardID,
		"name":     c.Name,
		"position": c.Position,
	}
}

func boardSummaryJSON(b models.Board) map[string]any {
	return map[string]any{
		"id":          b.ID,
		"name":        b.Name,
		"description": b.Description,
		"color":       b.Color,
		"archived":    b.Archived,
		"updated_at":  b.UpdatedAt.Format(time.RFC3339),
	}
}

func boardDetailJSON(b models.Board) map[string]any {
	out := boardSummaryJSON(b)
	out["created_at"] = b.CreatedAt.Format(time.RFC3339)
	return out
}

func tagJSON(t models.BoardTag) map[string]any {
	return map[string]any{
		"id":       t.ID,
		"name":     t.Name,
		"color":    t.Color,
		"position": t.Position,
	}
}
