package v1

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
	"github.com/rob121/kanban/internal/notifications"
	"github.com/rob121/kanban/internal/permissions"
	"github.com/rob121/kanban/internal/subscriptions"
)

type createCardRequest struct {
	BoardID     uint     `json:"board_id"`
	CategoryID  uint     `json:"category_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"`
	DueDate     *string  `json:"due_date"`
	AssigneeID  *uint    `json:"assignee_id"`
	TagIDs      []uint   `json:"tag_ids"`
}

type updateCardRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Priority    *string `json:"priority"`
	DueDate     *string `json:"due_date"`
	AssigneeID  *uint   `json:"assignee_id"`
	OwnerID     *uint   `json:"owner_id"`
	TagIDs      *[]uint `json:"tag_ids"`
}

type moveCardRequest struct {
	CategoryID uint `json:"category_id"`
	Position   int  `json:"position"`
}

type commentRequest struct {
	Body string `json:"body"`
}

func (h *Handler) ListCards(w http.ResponseWriter, r *http.Request) {
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

	query := database.DB.Preload("Assignee").Preload("Creator").Preload("Tags").
		Where("board_id = ?", boardID)

	switch r.URL.Query().Get("archived") {
	case "true":
		query = query.Where("archived = ?", true)
	case "all":
		// no filter
	default:
		query = query.Where("archived = ?", false)
	}

	var cards []models.Card
	query.Order("category_id asc, position asc").Find(&cards)

	items := make([]map[string]any, 0, len(cards))
	for _, c := range cards {
		items = append(items, cardJSON(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"cards": items})
}

func (h *Handler) ShowCard(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	cardID, err := pathUint(r, "id")
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var card models.Card
	if err := database.DB.Preload("Assignee").Preload("Creator").Preload("Tags").
		First(&card, cardID).Error; err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if _, ok := requireBoardAccess(w, r, user, card.BoardID, permissions.Access.CanView); !ok {
		return
	}

	writeJSON(w, http.StatusOK, cardJSON(card))
}

func (h *Handler) CreateCard(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createCardRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.BoardID == 0 || req.CategoryID == 0 || strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "board_id, category_id, and title are required")
		return
	}

	if _, ok := requireBoardAccess(w, r, user, req.BoardID, permissions.Access.CanCreate); !ok {
		return
	}

	var maxPos int
	database.DB.Model(&models.Card{}).Where("category_id = ?", req.CategoryID).
		Select("COALESCE(MAX(position), -1)").Scan(&maxPos)

	creatorID := user.ID
	card := models.Card{
		BoardID:     req.BoardID,
		CategoryID:  req.CategoryID,
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		Priority:    normalizePriority(req.Priority),
		Position:    maxPos + 1,
		CreatorID:   &creatorID,
		AssigneeID:  req.AssigneeID,
	}
	if req.DueDate != nil && *req.DueDate != "" {
		if t, err := time.Parse("2006-01-02", *req.DueDate); err == nil {
			card.DueDate = &t
		}
	}

	if err := database.DB.Omit("Creator", "Assignee", "Board", "Category", "Comments", "Tags").Create(&card).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not create card")
		return
	}
	syncCardTags(&card, req.BoardID, req.TagIDs)
	database.DB.Preload("Assignee").Preload("Creator").Preload("Tags").First(&card, card.ID)

	writeJSON(w, http.StatusCreated, cardJSON(card))
}

func (h *Handler) UpdateCard(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	cardID, err := pathUint(r, "id")
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var card models.Card
	if err := database.DB.First(&card, cardID).Error; err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if _, ok := requireBoardAccess(w, r, user, card.BoardID, permissions.Access.CanUpdate); !ok {
		return
	}

	var req updateCardRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title != nil {
		card.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		card.Description = *req.Description
	}
	if req.Priority != nil {
		card.Priority = normalizePriority(*req.Priority)
	}
	if req.DueDate != nil {
		if *req.DueDate == "" {
			card.DueDate = nil
		} else if t, err := time.Parse("2006-01-02", *req.DueDate); err == nil {
			card.DueDate = &t
		}
	}
	if req.AssigneeID != nil {
		if *req.AssigneeID == 0 {
			card.AssigneeID = nil
		} else {
			card.AssigneeID = req.AssigneeID
		}
	}
	if req.OwnerID != nil && card.IsOwnedBy(user.ID) {
		if participantIDs, err := permissions.BoardParticipantIDs(card.BoardID); err == nil {
			for _, pid := range participantIDs {
				if pid == *req.OwnerID {
					card.CreatorID = req.OwnerID
					break
				}
			}
		}
	}

	database.DB.Save(&card)
	if req.TagIDs != nil {
		syncCardTags(&card, card.BoardID, *req.TagIDs)
	}
	if card.AssigneeID != nil {
		subscriptions.ClearForUser(card.ID, *card.AssigneeID)
	}
	if card.CreatorID != nil {
		subscriptions.ClearForUser(card.ID, *card.CreatorID)
	}
	notifications.CardUpdated(card.ID, user.ID)

	database.DB.Preload("Assignee").Preload("Creator").Preload("Tags").First(&card, card.ID)
	writeJSON(w, http.StatusOK, cardJSON(card))
}

func (h *Handler) MoveCard(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	cardID, err := pathUint(r, "id")
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var card models.Card
	if err := database.DB.First(&card, cardID).Error; err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if _, ok := requireBoardAccess(w, r, user, card.BoardID, permissions.Access.CanMove); !ok {
		return
	}

	var req moveCardRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var cat models.Category
	if err := database.DB.Where("id = ? AND board_id = ?", req.CategoryID, card.BoardID).First(&cat).Error; err != nil {
		writeError(w, http.StatusBadRequest, "invalid category")
		return
	}

	oldCategoryID := card.CategoryID
	card.CategoryID = req.CategoryID
	card.Position = req.Position

	if cat.Position >= 2 && card.AssigneeID == nil {
		card.AssigneeID = &user.ID
	}

	database.DB.Save(&card)
	if card.AssigneeID != nil {
		subscriptions.ClearForUser(card.ID, *card.AssigneeID)
	}
	reorderCards(oldCategoryID)
	reorderCards(card.CategoryID)

	if oldCategoryID != card.CategoryID {
		notifications.CardMoved(card.ID, user.ID, oldCategoryID)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ArchiveCard(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	cardID, err := pathUint(r, "id")
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var card models.Card
	if err := database.DB.First(&card, cardID).Error; err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if _, ok := requireBoardAccess(w, r, user, card.BoardID, permissions.Access.CanDelete); !ok {
		return
	}

	card.Archived = true
	database.DB.Save(&card)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AddComment(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	cardID, err := pathUint(r, "id")
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var card models.Card
	if err := database.DB.First(&card, cardID).Error; err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if _, ok := requireBoardAccess(w, r, user, card.BoardID, permissions.Access.CanView); !ok {
		return
	}

	var req commentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}

	comment := models.Comment{
		CardID: card.ID,
		UserID: user.ID,
		Body:   body,
	}
	database.DB.Create(&comment)
	database.DB.Preload("User").First(&comment, comment.ID)
	notifications.CommentAdded(card.ID, comment.ID, user.ID)

	writeJSON(w, http.StatusCreated, commentJSON(comment))
}

func cardJSON(c models.Card) map[string]any {
	out := map[string]any{
		"id":          c.ID,
		"board_id":    c.BoardID,
		"category_id": c.CategoryID,
		"title":       c.Title,
		"description": c.Description,
		"priority":    c.Priority,
		"position":    c.Position,
		"archived":    c.Archived,
		"created_at":  c.CreatedAt.Format(time.RFC3339),
		"updated_at":  c.UpdatedAt.Format(time.RFC3339),
	}
	if c.DueDate != nil {
		out["due_date"] = c.DueDate.Format("2006-01-02")
	}
	if c.AssigneeID != nil {
		out["assignee_id"] = *c.AssigneeID
	}
	if c.CreatorID != nil {
		out["owner_id"] = *c.CreatorID
	}
	if c.Assignee != nil {
		out["assignee"] = userJSON(*c.Assignee)
	}
	if c.Creator != nil {
		out["owner"] = userJSON(*c.Creator)
	}
	if len(c.Tags) > 0 {
		tags := make([]map[string]any, 0, len(c.Tags))
		for _, t := range c.Tags {
			tags = append(tags, tagJSON(t))
		}
		out["tags"] = tags
	}
	return out
}

func commentJSON(c models.Comment) map[string]any {
	out := map[string]any{
		"id":         c.ID,
		"card_id":    c.CardID,
		"user_id":    c.UserID,
		"body":       c.Body,
		"created_at": c.CreatedAt.Format(time.RFC3339),
	}
	if c.User.ID != 0 {
		out["user"] = userJSON(c.User)
	}
	return out
}

func normalizePriority(priority string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "high":
		return "high"
	case "low":
		return "low"
	default:
		return "medium"
	}
}

func syncCardTags(card *models.Card, boardID uint, tagIDs []uint) {
	var tags []models.BoardTag
	seen := make(map[uint]bool)
	for _, id := range tagIDs {
		if id == 0 || seen[id] {
			continue
		}
		var tag models.BoardTag
		if err := database.DB.Where("id = ? AND board_id = ?", id, boardID).First(&tag).Error; err != nil {
			continue
		}
		tags = append(tags, tag)
		seen[id] = true
	}
	_ = database.DB.Model(card).Association("Tags").Replace(tags)
}

func reorderCards(categoryID uint) {
	var cards []models.Card
	database.DB.Where("category_id = ? AND archived = ?", categoryID, false).
		Order("position asc, updated_at asc").
		Find(&cards)
	for i, c := range cards {
		if c.Position != i {
			database.DB.Model(&c).Update("position", i)
		}
	}
}

// parseUint kept for any string conversions in tests
func parseUint(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}
