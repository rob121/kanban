package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/rob121/kanban/internal/auth"
	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
	"github.com/rob121/kanban/internal/permissions"
	"gorm.io/gorm"
)

type MemberHandler struct {
	Render *Renderer
}

func (h *MemberHandler) Manage(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	boardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	access, err := permissions.GetAccess(user, boardID)
	if err != nil || !access.CanManageMembers() {
		http.NotFound(w, r)
		return
	}

	var members []models.BoardMember
	database.DB.Preload("User").Where("board_id = ?", boardID).Order("created_at asc").Find(&members)

	memberIDs := map[uint]bool{access.Board.UserID: true}
	for _, m := range members {
		memberIDs[m.UserID] = true
	}

	var allUsers []models.User
	database.DB.Order("name asc").Find(&allUsers)

	var available []models.User
	for _, u := range allUsers {
		if !memberIDs[u.ID] {
			available = append(available, u)
		}
	}

	var owner models.User
	_ = database.DB.First(&owner, access.Board.UserID).Error

	_ = h.Render.Render(w, "boards/members.html", buildPage(w, r, access.Board.Name+" · Members", user, map[string]any{
		"Board":          access.Board,
		"Owner":          owner,
		"Members":        members,
		"AvailableUsers": available,
		"OwnerID":        access.Board.UserID,
	}))
}

func (h *MemberHandler) Add(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	boardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	access, err := permissions.GetAccess(user, boardID)
	if err != nil || !access.CanManageMembers() {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	memberUserID, err := parseUserIDForm(r)
	if err != nil || memberUserID == 0 {
		http.Error(w, "user required", http.StatusBadRequest)
		return
	}
	if memberUserID == access.Board.UserID {
		http.Redirect(w, r, "/boards/"+strconv.FormatUint(uint64(boardID), 10)+"/members", http.StatusSeeOther)
		return
	}

	canCreate, canUpdate, canDelete, canMove, canAttach := parseMemberPermissions(r)
	member := models.BoardMember{
		BoardID:   boardID,
		UserID:    memberUserID,
		CanCreate: canCreate,
		CanUpdate: canUpdate,
		CanDelete: canDelete,
		CanMove:   canMove,
		CanAttach: canAttach,
	}

	var existing models.BoardMember
	err = database.DB.Where("board_id = ? AND user_id = ?", boardID, memberUserID).First(&existing).Error
	if err == nil {
		existing.CanCreate = canCreate
		existing.CanUpdate = canUpdate
		existing.CanDelete = canDelete
		existing.CanMove = canMove
		existing.CanAttach = canAttach
		database.DB.Save(&existing)
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		database.DB.Create(&member)
	} else {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/boards/"+strconv.FormatUint(uint64(boardID), 10)+"/members", http.StatusSeeOther)
}

func (h *MemberHandler) Update(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	boardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	memberUserID, err := pathUint(r, "userId")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	access, err := permissions.GetAccess(user, boardID)
	if err != nil || !access.CanManageMembers() {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var member models.BoardMember
	if err := database.DB.Where("board_id = ? AND user_id = ?", boardID, memberUserID).First(&member).Error; err != nil {
		http.NotFound(w, r)
		return
	}

	member.CanCreate, member.CanUpdate, member.CanDelete, member.CanMove, member.CanAttach = parseMemberPermissions(r)
	database.DB.Save(&member)

	http.Redirect(w, r, "/boards/"+strconv.FormatUint(uint64(boardID), 10)+"/members", http.StatusSeeOther)
}

func (h *MemberHandler) Remove(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.GetUser(r)
	boardID, err := pathUint(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	memberUserID, err := pathUint(r, "userId")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	access, err := permissions.GetAccess(user, boardID)
	if err != nil || !access.CanManageMembers() {
		http.NotFound(w, r)
		return
	}

	database.DB.Where("board_id = ? AND user_id = ?", boardID, memberUserID).Delete(&models.BoardMember{})
	http.Redirect(w, r, "/boards/"+strconv.FormatUint(uint64(boardID), 10)+"/members", http.StatusSeeOther)
}
