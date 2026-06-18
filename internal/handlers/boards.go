package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/rob121/kanban/internal/auth"
	boardcleanup "github.com/rob121/kanban/internal/boards"
	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
	"github.com/rob121/kanban/internal/permissions"
)

type BoardHandler struct {
	Render *Renderer
}

type BoardListData struct {
	Boards []models.Board
}

func (h *BoardHandler) Index(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	boards, err := permissions.AccessibleBoards(user)
	if err != nil {
		http.Error(w, "could not load boards", http.StatusInternalServerError)
		return
	}

	_ = h.Render.Render(w, "boards/index.html", buildPage(w, r, "My Boards", user, BoardListData{Boards: boards}))
}

func (h *BoardHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	if r.Method == http.MethodGet {
		_ = h.Render.Render(w, "boards/new.html", buildPage(w, r, "New Board", user, nil))
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		_ = h.Render.Render(w, "boards/new.html", buildPageError(w, r, "New Board", user, nil, "Board name is required"))
		return
	}

	board := models.Board{
		UserID:      user.ID,
		Name:        name,
		Description: r.FormValue("description"),
		Color:       normalizeBoardColor(r.FormValue("color")),
	}
	if err := database.DB.Create(&board).Error; err != nil {
		http.Error(w, "could not create board", http.StatusInternalServerError)
		return
	}
	if err := models.SeedDefaultCategories(database.DB, board.ID); err != nil {
		http.Error(w, "could not seed categories", http.StatusInternalServerError)
		return
	}
	if err := models.SeedDefaultTags(database.DB, board.ID); err != nil {
		http.Error(w, "could not seed tags", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/boards/"+strconv.FormatUint(uint64(board.ID), 10), http.StatusSeeOther)
}

func (h *BoardHandler) Show(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	boardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	access, categories, cards, err := loadBoardView(user, boardID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	participants, _ := permissions.BoardParticipants(boardID)
	boardTags := boardTagsOrSeed(boardID)

	_ = h.Render.Render(w, "boards/show.html", buildPage(w, r, access.Board.Name, user, map[string]any{
		"Board":        access.Board,
		"Access":       access,
		"Categories":   categories,
		"Cards":        cardsByCategory(cards),
		"Participants": participants,
		"BoardTags":    boardTags,
	}))
}

func (h *BoardHandler) Archive(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	boardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	access, err := permissions.GetAccess(user, boardID)
	if err != nil || !access.CanManageBoard() || access.Board.Archived {
		http.NotFound(w, r)
		return
	}

	access.Board.Archived = true
	if err := database.DB.Save(&access.Board).Error; err != nil {
		http.Error(w, "could not archive board", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/boards", http.StatusSeeOther)
}

func (h *BoardHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	if user == nil || !user.IsAdmin {
		http.NotFound(w, r)
		return
	}

	boardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var board models.Board
	if err := database.DB.Unscoped().First(&board, boardID).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	if err := boardcleanup.HardDelete(board.ID); err != nil {
		http.Error(w, "could not delete board", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/boards", http.StatusSeeOther)
}

func (h *BoardHandler) Settings(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	boardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	access, err := permissions.GetAccess(user, boardID)
	if err != nil || !access.CanManageBoard() {
		http.NotFound(w, r)
		return
	}

	board := access.Board

	if r.Method == http.MethodGet {
		_ = h.Render.Render(w, "boards/settings.html", buildPage(w, r, board.Name+" · Settings", user, map[string]any{
			"Board":  board,
			"Access": access,
		}))
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		_ = h.Render.Render(w, "boards/settings.html", buildPageError(w, r, board.Name+" · Settings", user, map[string]any{
			"Board":  board,
			"Access": access,
		}, "Board name is required"))
		return
	}

	board.Name = name
	board.Description = r.FormValue("description")
	board.Color = normalizeBoardColor(r.FormValue("color"))

	if err := database.DB.Save(&board).Error; err != nil {
		http.Error(w, "could not save board", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/boards/"+strconv.FormatUint(uint64(board.ID), 10), http.StatusSeeOther)
}

func loadBoardView(user *models.User, boardID uint) (permissions.Access, []models.Category, []models.Card, error) {
	access, err := permissions.GetAccess(user, boardID)
	if err != nil {
		return access, nil, nil, err
	}
	if !access.CanView() {
		return access, nil, nil, errors.New("no access")
	}

	var categories []models.Category
	database.DB.Where("board_id = ?", boardID).Order("position asc").Find(&categories)

	var cards []models.Card
	database.DB.Preload("Assignee").Preload("Creator").Preload("Comments.User").Preload("Tags").
		Where("board_id = ? AND archived = ?", boardID, false).
		Order("position asc").
		Find(&cards)

	return access, categories, cards, nil
}

func cardsByCategory(cards []models.Card) map[uint][]models.Card {
	out := make(map[uint][]models.Card)
	for _, c := range cards {
		out[c.CategoryID] = append(out[c.CategoryID], c)
	}
	return out
}

func requireBoardPerm(w http.ResponseWriter, r *http.Request, user *models.User, boardID uint, check func(permissions.Access) bool) (permissions.Access, bool) {
	access, err := permissions.GetAccess(user, boardID)
	if err != nil || !check(access) {
		http.NotFound(w, r)
		return access, false
	}
	return access, true
}
