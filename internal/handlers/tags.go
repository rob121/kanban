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

type TagHandler struct {
	Render *Renderer
}

type tagListItem struct {
	models.BoardTag
	ActiveCards int64
}

func activeCardCountForTag(tagID, boardID uint) int64 {
	var count int64
	database.DB.Table("card_tags").
		Joins("INNER JOIN cards ON cards.id = card_tags.card_id AND cards.deleted_at IS NULL").
		Where("card_tags.board_tag_id = ? AND cards.board_id = ? AND cards.archived = ?", tagID, boardID, false).
		Count(&count)
	return count
}

func (h *TagHandler) Manage(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	boardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	access, err := permissions.GetAccess(user, boardID)
	if err != nil || !access.CanManageTags() {
		http.NotFound(w, r)
		return
	}

	var tags []models.BoardTag
	database.DB.Where("board_id = ?", boardID).Order("position asc").Find(&tags)

	tagItems := make([]tagListItem, 0, len(tags))
	for _, t := range tags {
		tagItems = append(tagItems, tagListItem{
			BoardTag:    t,
			ActiveCards: activeCardCountForTag(t.ID, boardID),
		})
	}

	page := buildPage(w, r, access.Board.Name+" · Tags", user, map[string]any{
		"Board": access.Board,
		"Tags":  tagItems,
	})
	if r.URL.Query().Get("error") == "in_use" {
		page.Error = "Cannot remove a tag that is on active cards. Archive those cards or remove the tag from them first."
	}

	_ = h.Render.Render(w, "boards/tags.html", page)
}

func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	boardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if _, ok := requireBoardPerm(w, r, user, boardID, permissions.Access.CanManageTags); !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	color := normalizeTagColor(r.FormValue("color"))
	if name == "" {
		http.Redirect(w, r, "/boards/"+strconv.FormatUint(uint64(boardID), 10)+"/tags", http.StatusSeeOther)
		return
	}

	var maxPos int
	database.DB.Model(&models.BoardTag{}).Where("board_id = ?", boardID).
		Select("COALESCE(MAX(position), -1)").Scan(&maxPos)

	database.DB.Create(&models.BoardTag{
		BoardID:  boardID,
		Name:     name,
		Color:    color,
		Position: maxPos + 1,
	})

	http.Redirect(w, r, "/boards/"+strconv.FormatUint(uint64(boardID), 10)+"/tags", http.StatusSeeOther)
}

func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	boardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	tagID, err := pathUint(r, "tagId")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if _, ok := requireBoardPerm(w, r, user, boardID, permissions.Access.CanManageTags); !ok {
		return
	}

	var tag models.BoardTag
	if err := database.DB.Where("id = ? AND board_id = ?", tagID, boardID).First(&tag).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	if activeCardCountForTag(tag.ID, boardID) > 0 {
		http.Redirect(w, r, "/boards/"+strconv.FormatUint(uint64(boardID), 10)+"/tags?error=in_use", http.StatusSeeOther)
		return
	}

	database.DB.Table("card_tags").Where("board_tag_id = ?", tag.ID).Delete(nil)
	database.DB.Delete(&tag)

	http.Redirect(w, r, "/boards/"+strconv.FormatUint(uint64(boardID), 10)+"/tags", http.StatusSeeOther)
}

func loadBoardTags(boardID uint) []models.BoardTag {
	var tags []models.BoardTag
	database.DB.Where("board_id = ?", boardID).Order("position asc").Find(&tags)
	return tags
}

func boardTagsOrSeed(boardID uint) []models.BoardTag {
	tags := loadBoardTags(boardID)
	if len(tags) == 0 {
		_ = models.SeedDefaultTags(database.DB, boardID)
		tags = loadBoardTags(boardID)
	}
	return tags
}

func syncCardTags(card *models.Card, boardID uint, tagIDStrs []string) {
	var tags []models.BoardTag
	seen := make(map[uint]bool)
	for _, s := range tagIDStrs {
		id, err := strconv.ParseUint(s, 10, 64)
		if err != nil || id == 0 || seen[uint(id)] {
			continue
		}
		var tag models.BoardTag
		if err := database.DB.Where("id = ? AND board_id = ?", id, boardID).First(&tag).Error; err != nil {
			continue
		}
		tags = append(tags, tag)
		seen[uint(id)] = true
	}
	_ = database.DB.Model(card).Association("Tags").Replace(tags)
}

func selectedTagSet(tags []models.BoardTag) map[string]bool {
	out := make(map[string]bool, len(tags))
	for _, t := range tags {
		out[strconv.FormatUint(uint64(t.ID), 10)] = true
	}
	return out
}

func normalizeTagColor(color string) string {
	switch strings.ToLower(strings.TrimSpace(color)) {
	case "primary", "danger", "warning", "info", "success", "secondary":
		return strings.ToLower(color)
	default:
		return "primary"
	}
}
