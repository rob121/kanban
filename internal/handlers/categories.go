package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/rob121/kanban/internal/auth"
	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
	"github.com/rob121/kanban/internal/permissions"
)

type CategoryHandler struct {
	Render *Renderer
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	boardID, _ := strconv.ParseUint(r.FormValue("board_id"), 10, 64)
	if _, ok := requireBoardPerm(w, r, user, uint(boardID), permissions.Access.CanManageBoard); !ok {
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

	var maxPos int
	database.DB.Model(&models.Category{}).
		Where("board_id = ?", boardID).
		Select("COALESCE(MAX(position), -1)").
		Scan(&maxPos)

	cat := models.Category{
		BoardID:  uint(boardID),
		Name:     name,
		Position: maxPos + 1,
	}
	database.DB.Create(&cat)

	http.Redirect(w, r, "/boards/"+strconv.FormatUint(boardID, 10), http.StatusSeeOther)
}

func (h *CategoryHandler) Move(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	categoryID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var cat models.Category
	if err := database.DB.First(&cat, categoryID).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	if _, ok := requireBoardPerm(w, r, user, cat.BoardID, permissions.Access.CanManageBoard); !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	position, _ := strconv.Atoi(r.FormValue("position"))
	cat.Position = position
	database.DB.Save(&cat)
	reorderCategories(cat.BoardID)

	w.WriteHeader(http.StatusNoContent)
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	categoryID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var cat models.Category
	if err := database.DB.First(&cat, categoryID).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	if _, ok := requireBoardPerm(w, r, user, cat.BoardID, permissions.Access.CanDeleteColumn); !ok {
		return
	}

	var count int64
	database.DB.Model(&models.Category{}).Where("board_id = ?", cat.BoardID).Count(&count)
	if count <= 1 {
		http.Error(w, "cannot delete the only column on a board", http.StatusBadRequest)
		return
	}

	var cardCount int64
	database.DB.Model(&models.Card{}).Where("category_id = ?", cat.ID).Count(&cardCount)
	if cardCount > 0 {
		http.Error(w, "cannot delete a column that contains cards; move or archive them first", http.StatusBadRequest)
		return
	}

	database.DB.Delete(&cat)
	reorderCategories(cat.BoardID)

	w.WriteHeader(http.StatusNoContent)
}

func reorderCategories(boardID uint) {
	var cats []models.Category
	database.DB.Where("board_id = ?", boardID).
		Order("position asc, updated_at asc").
		Find(&cats)
	for i, c := range cats {
		if c.Position != i {
			database.DB.Model(&c).Update("position", i)
		}
	}
}
