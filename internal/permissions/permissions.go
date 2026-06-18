package permissions

import (
	"errors"

	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
	"gorm.io/gorm"
)

var ErrNoAccess = errors.New("no access to board")

// Access describes what a user may do on a board.
type Access struct {
	Board   models.Board
	IsOwner bool
	IsAdmin bool
	Member  *models.BoardMember
}

func (a Access) CanView() bool {
	return a.IsOwner || a.IsAdmin || a.Member != nil
}

func (a Access) CanCreate() bool {
	if a.IsOwner || a.IsAdmin {
		return true
	}
	return a.Member != nil && a.Member.CanCreate
}

func (a Access) CanUpdate() bool {
	if a.IsOwner || a.IsAdmin {
		return true
	}
	return a.Member != nil && a.Member.CanUpdate
}

func (a Access) CanDelete() bool {
	if a.IsOwner || a.IsAdmin {
		return true
	}
	return a.Member != nil && a.Member.CanDelete
}

func (a Access) CanMove() bool {
	if a.IsOwner || a.IsAdmin {
		return true
	}
	return a.Member != nil && a.Member.CanMove
}

func (a Access) CanAttach() bool {
	if a.IsOwner || a.IsAdmin {
		return true
	}
	return a.Member != nil && a.Member.CanAttach
}

func (a Access) CanRemoveAttachment(uploaderID, userID uint) bool {
	if a.IsOwner || a.IsAdmin || a.CanDelete() {
		return true
	}
	return a.CanAttach() && uploaderID == userID
}

func (a Access) CanManageBoard() bool {
	return a.IsOwner || a.IsAdmin
}

func (a Access) CanManageMembers() bool {
	return a.IsOwner || a.IsAdmin
}

func (a Access) CanDeleteColumn() bool {
	return a.IsAdmin
}

func GetAccess(user *models.User, boardID uint) (Access, error) {
	var board models.Board
	if err := database.DB.First(&board, boardID).Error; err != nil {
		return Access{}, err
	}

	access := Access{Board: board}
	if board.Archived && (user == nil || !user.IsAdmin) {
		return Access{}, ErrNoAccess
	}
	if user.IsAdmin {
		access.IsAdmin = true
		access.IsOwner = board.UserID == user.ID
		return access, nil
	}
	if board.UserID == user.ID {
		access.IsOwner = true
		return access, nil
	}

	var member models.BoardMember
	err := database.DB.Where("board_id = ? AND user_id = ?", boardID, user.ID).First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Access{}, ErrNoAccess
		}
		return Access{}, err
	}
	access.Member = &member
	return access, nil
}

func AccessibleBoards(user *models.User) ([]models.Board, error) {
	if user.IsAdmin {
		var boards []models.Board
		err := database.DB.Where("archived = ?", false).Order("updated_at desc").Find(&boards).Error
		return boards, err
	}

	var boards []models.Board
	err := database.DB.
		Distinct("boards.*").
		Joins("LEFT JOIN board_members ON board_members.board_id = boards.id AND board_members.user_id = ?", user.ID).
		Where("(boards.user_id = ? OR board_members.user_id = ?) AND boards.archived = ?", user.ID, user.ID, false).
		Order("boards.updated_at desc").
		Find(&boards).Error
	return boards, err
}

func BoardParticipants(boardID uint) ([]models.User, error) {
	var board models.Board
	if err := database.DB.Preload("User").First(&board, boardID).Error; err != nil {
		return nil, err
	}

	var members []models.BoardMember
	if err := database.DB.Preload("User").Where("board_id = ?", boardID).Find(&members).Error; err != nil {
		return nil, err
	}

	users := []models.User{board.User}
	seen := map[uint]bool{board.UserID: true}
	for _, m := range members {
		if !seen[m.UserID] {
			users = append(users, m.User)
			seen[m.UserID] = true
		}
	}
	return users, nil
}

func BoardParticipantIDs(boardID uint) ([]uint, error) {
	users, err := BoardParticipants(boardID)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, len(users))
	for i, u := range users {
		ids[i] = u.ID
	}
	return ids, nil
}

func MemberFromForm(canCreate, canUpdate, canDelete, canMove, canAttach bool) models.BoardMember {
	return models.BoardMember{
		CanCreate: canCreate,
		CanUpdate: canUpdate,
		CanDelete: canDelete,
		CanMove:   canMove,
		CanAttach: canAttach,
	}
}
