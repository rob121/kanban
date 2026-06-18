package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rob121/kanban/internal/auth"
	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
	"github.com/rob121/kanban/internal/notifications"
	"github.com/rob121/kanban/internal/permissions"
	"github.com/rob121/kanban/internal/subscriptions"
)

type CardHandler struct {
	Render *Renderer
}

const commentsPerPage = 5

func loadCardComments(cardID uint, page int) ([]models.Comment, int64, int, int) {
	var total int64
	database.DB.Model(&models.Comment{}).Where("card_id = ?", cardID).Count(&total)

	totalPages := int((total + commentsPerPage - 1) / commentsPerPage)
	if totalPages == 0 {
		totalPages = 1
	}
	if page <= 0 {
		page = totalPages
	}
	if page > totalPages {
		page = totalPages
	}

	var comments []models.Comment
	offset := (page - 1) * commentsPerPage
	database.DB.Where("card_id = ?", cardID).
		Order("created_at asc").
		Preload("User").
		Limit(commentsPerPage).
		Offset(offset).
		Find(&comments)

	return comments, total, page, totalPages
}

func preloadKanbanCard(card *models.Card) {
	database.DB.Preload("Assignee").Preload("Creator").Preload("Tags").Preload("Comments").First(card, card.ID)
}

func (h *CardHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	if err := auth.ParseRequestForm(r); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	boardID, _ := strconv.ParseUint(r.FormValue("board_id"), 10, 64)
	categoryID, _ := strconv.ParseUint(r.FormValue("category_id"), 10, 64)

	if boardID == 0 || categoryID == 0 {
		http.Error(w, "board and category are required", http.StatusBadRequest)
		return
	}

	if _, ok := requireBoardPerm(w, r, user, uint(boardID), permissions.Access.CanCreate); !ok {
		return
	}

	var maxPos int
	database.DB.Model(&models.Card{}).
		Where("category_id = ?", categoryID).
		Select("COALESCE(MAX(position), -1)").
		Scan(&maxPos)

	creatorID := user.ID
	card := models.Card{
		BoardID:     uint(boardID),
		CategoryID:  uint(categoryID),
		Title:       strings.TrimSpace(r.FormValue("title")),
		Description: r.FormValue("description"),
		Priority:    defaultPriority(r.FormValue("priority")),
		Position:    maxPos + 1,
		CreatorID:   &creatorID,
	}
	if card.Title == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}
	if dd := r.FormValue("due_date"); dd != "" {
		if t, err := time.Parse("2006-01-02", dd); err == nil {
			card.DueDate = &t
		}
	}

	if err := database.DB.Omit("Creator", "Assignee", "Board", "Category", "Comments", "Tags").Create(&card).Error; err != nil {
		http.Error(w, "could not create card", http.StatusInternalServerError)
		return
	}
	syncCardTags(&card, uint(boardID), r.Form["tag_ids"])

	if isTurbo(r) {
		preloadKanbanCard(&card)
		w.Header().Set("Content-Type", "text/vnd.turbo-stream.html; charset=utf-8")
		_ = h.Render.RenderPartial(w, "card", card)
		return
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h *CardHandler) Show(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	cardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var card models.Card
	if err := database.DB.Preload("Assignee").Preload("Creator").Preload("Tags").
		First(&card, cardID).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	access, ok := requireBoardPerm(w, r, user, card.BoardID, permissions.Access.CanView)
	if !ok {
		return
	}

	if wantsPartial(r) {
		if r.URL.Query().Get("view") == "card" {
			preloadKanbanCard(&card)
			_ = h.Render.RenderPartial(w, "card", card)
			return
		}

		commentsPage := 0
		if p, err := strconv.Atoi(r.URL.Query().Get("comments_page")); err == nil && p > 0 {
			commentsPage = p
		}
		comments, commentsTotal, commentsPage, commentsPages := loadCardComments(cardID, commentsPage)
		card.Comments = comments
		card.Attachments = loadCardAttachments(cardID)

		participants, _ := permissions.BoardParticipants(card.BoardID)
		boardTags := boardTagsOrSeed(card.BoardID)
		_ = h.Render.RenderPartial(w, "card-detail", map[string]any{
			"Card":                 card,
			"User":                 user,
			"Participants":         participants,
			"Access":               access,
			"BoardTags":            boardTags,
			"SelectedTags":         selectedTagSet(card.Tags),
			"CommentsTotal":        commentsTotal,
			"CommentsPage":         commentsPage,
			"CommentsPages":        commentsPages,
			"CanTransferOwnership": card.IsOwnedBy(user.ID),
			"CanSubscribe":         subscriptions.CanSubscribe(card, user.ID),
			"IsSubscribed":         subscriptions.IsSubscribed(card.ID, user.ID),
		})
		return
	}

	http.Redirect(w, r, "/boards/"+strconv.FormatUint(uint64(card.BoardID), 10), http.StatusSeeOther)
}

func (h *CardHandler) Update(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	cardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var card models.Card
	if err := database.DB.First(&card, cardID).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	if _, ok := requireBoardPerm(w, r, user, card.BoardID, permissions.Access.CanUpdate); !ok {
		return
	}

	if err := auth.ParseRequestForm(r); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	card.Title = strings.TrimSpace(r.FormValue("title"))
	card.Description = r.FormValue("description")
	card.Priority = defaultPriority(r.FormValue("priority"))

	if dd := r.FormValue("due_date"); dd != "" {
		if t, err := time.Parse("2006-01-02", dd); err == nil {
			card.DueDate = &t
		}
	} else {
		card.DueDate = nil
	}

	if aid := r.FormValue("assignee_id"); aid != "" {
		if id, err := strconv.ParseUint(aid, 10, 64); err == nil && id > 0 {
			uid := uint(id)
			card.AssigneeID = &uid
		}
	} else {
		card.AssigneeID = nil
	}

	if card.IsOwnedBy(user.ID) {
		if oid := r.FormValue("owner_id"); oid != "" {
			if id, err := strconv.ParseUint(oid, 10, 64); err == nil && id > 0 {
				if participantIDs, err := permissions.BoardParticipantIDs(card.BoardID); err == nil {
					uid := uint(id)
					for _, pid := range participantIDs {
						if pid == uid {
							card.CreatorID = &uid
							break
						}
					}
				}
			}
		}
	}

	database.DB.Save(&card)
	syncCardTags(&card, card.BoardID, r.Form["tag_ids"])

	if card.AssigneeID != nil {
		subscriptions.ClearForUser(card.ID, *card.AssigneeID)
	}
	if card.CreatorID != nil {
		subscriptions.ClearForUser(card.ID, *card.CreatorID)
	}

	notifications.CardUpdated(card.ID, user.ID)

	if wantsPartial(r) {
		preloadKanbanCard(&card)
		_ = h.Render.RenderPartial(w, "card", card)
		return
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h *CardHandler) Move(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	cardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var card models.Card
	if err := database.DB.First(&card, cardID).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	if _, ok := requireBoardPerm(w, r, user, card.BoardID, permissions.Access.CanMove); !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	categoryID, _ := strconv.ParseUint(r.FormValue("category_id"), 10, 64)
	position, _ := strconv.Atoi(r.FormValue("position"))

	var cat models.Category
	if err := database.DB.Where("id = ? AND board_id = ?", categoryID, card.BoardID).First(&cat).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	oldCategoryID := card.CategoryID
	card.CategoryID = uint(categoryID)
	card.Position = position

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

func (h *CardHandler) Archive(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	cardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var card models.Card
	if err := database.DB.First(&card, cardID).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	if _, ok := requireBoardPerm(w, r, user, card.BoardID, permissions.Access.CanDelete); !ok {
		return
	}

	card.Archived = true
	database.DB.Save(&card)
	w.WriteHeader(http.StatusNoContent)
}

func (h *CardHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	cardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var card models.Card
	if err := database.DB.First(&card, cardID).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	if _, ok := requireBoardPerm(w, r, user, card.BoardID, permissions.Access.CanView); !ok {
		return
	}

	if err := auth.ParseRequestForm(r); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	body := strings.TrimSpace(r.FormValue("body"))
	if body == "" {
		http.Error(w, "comment required", http.StatusBadRequest)
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

	if wantsPartial(r) {
		_ = h.Render.RenderPartial(w, "comment", comment)
		return
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h *CardHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	cardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var card models.Card
	if err := database.DB.First(&card, cardID).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	if _, ok := requireBoardPerm(w, r, user, card.BoardID, permissions.Access.CanView); !ok {
		return
	}
	if !subscriptions.CanSubscribe(card, user.ID) {
		http.Error(w, "cannot subscribe to this card", http.StatusBadRequest)
		return
	}
	if err := subscriptions.Subscribe(card.ID, user.ID); err != nil {
		http.Error(w, "could not subscribe", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CardHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	cardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var card models.Card
	if err := database.DB.First(&card, cardID).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	if _, ok := requireBoardPerm(w, r, user, card.BoardID, permissions.Access.CanView); !ok {
		return
	}
	if err := subscriptions.Unsubscribe(card.ID, user.ID); err != nil {
		http.Error(w, "could not unsubscribe", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

func defaultPriority(p string) string {
	switch strings.ToLower(p) {
	case "low", "medium", "high":
		return strings.ToLower(p)
	default:
		return "medium"
	}
}

func isTurbo(r *http.Request) bool {
	return r.Header.Get("Accept") == "text/vnd.turbo-stream.html" ||
		r.Header.Get("Turbo-Frame") != "" ||
		strings.Contains(r.Header.Get("Accept"), "text/vnd.turbo-stream.html")
}

func wantsPartial(r *http.Request) bool {
	return isTurbo(r) || r.URL.Query().Get("partial") == "1" || r.Header.Get("X-Partial") == "1"
}

func pathUint(r *http.Request, key string) (uint, error) {
	val := r.PathValue(key)
	if val == "" {
		val = r.URL.Query().Get(key)
	}
	id, err := strconv.ParseUint(val, 10, 64)
	return uint(id), err
}
