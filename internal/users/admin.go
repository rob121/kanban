package users

import (
	"errors"
	"fmt"

	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
	"github.com/rob121/kanban/internal/storage"
)

var (
	ErrAssignedCards   = errors.New("user is assigned to one or more cards")
	ErrActiveAdmin     = errors.New("cannot remove the only active administrator")
	ErrSelfAction      = errors.New("cannot archive or delete your own account")
	ErrAlreadyArchived = errors.New("user is already archived")
)

func AssignedCardCount(userID uint) (int64, error) {
	var count int64
	err := database.DB.Model(&models.Card{}).Where("assignee_id = ?", userID).Count(&count).Error
	return count, err
}

func OwnedBoardCount(userID uint) (int64, error) {
	var count int64
	err := database.DB.Model(&models.Board{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func ActiveAdminCount(excludeUserID uint) (int64, error) {
	var count int64
	err := database.DB.Model(&models.User{}).
		Where("is_admin = ? AND archived = ? AND id <> ?", true, false, excludeUserID).
		Count(&count).Error
	return count, err
}

func CanHardDelete(userID uint) (bool, string) {
	assigned, err := AssignedCardCount(userID)
	if err != nil {
		return false, "could not check assigned cards"
	}
	if assigned > 0 {
		return false, fmt.Sprintf("assigned to %d card(s); reassign those cards first", assigned)
	}

	return true, ""
}

func Archive(actorID, userID uint) error {
	if actorID == userID {
		return ErrSelfAction
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return err
	}
	if user.Archived {
		return ErrAlreadyArchived
	}

	if user.IsAdmin {
		others, err := ActiveAdminCount(userID)
		if err != nil {
			return err
		}
		if others == 0 {
			return ErrActiveAdmin
		}
	}

	user.Archived = true
	return database.DB.Save(&user).Error
}

func HardDelete(actorID, userID uint) error {
	if actorID == userID {
		return ErrSelfAction
	}

	if ok, reason := CanHardDelete(userID); !ok {
		return errors.New(reason)
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return err
	}

	if user.IsAdmin {
		others, err := ActiveAdminCount(userID)
		if err != nil {
			return err
		}
		if others == 0 {
			return ErrActiveAdmin
		}
	}

	var attachments []models.CardAttachment
	if err := database.DB.Where("user_id = ?", userID).Find(&attachments).Error; err != nil {
		return err
	}
	for _, attachment := range attachments {
		_ = storage.DeleteAttachment(attachment.StoredName)
	}

	if err := database.DB.Where("user_id = ?", userID).Delete(&models.BoardMember{}).Error; err != nil {
		return err
	}
	if err := database.DB.Where("user_id = ?", userID).Delete(&models.CardSubscriber{}).Error; err != nil {
		return err
	}
	if err := database.DB.Where("user_id = ?", userID).Delete(&models.Comment{}).Error; err != nil {
		return err
	}
	if err := database.DB.Where("user_id = ?", userID).Delete(&models.CardAttachment{}).Error; err != nil {
		return err
	}
	if err := database.DB.Where("user_id = ?", userID).Delete(&models.APIToken{}).Error; err != nil {
		return err
	}
	if err := database.DB.Model(&models.Card{}).Where("creator_id = ?", userID).Update("creator_id", nil).Error; err != nil {
		return err
	}
	if err := database.DB.Model(&models.Board{}).Where("user_id = ?", userID).Update("user_id", actorID).Error; err != nil {
		return err
	}

	return database.DB.Delete(&user).Error
}
