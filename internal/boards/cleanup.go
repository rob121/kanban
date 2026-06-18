package boards

import (
	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
	"github.com/rob121/kanban/internal/storage"
)

// HardDelete permanently removes a board and all related data, including attachment files.
func HardDelete(boardID uint) error {
	var attachments []models.CardAttachment
	if err := database.DB.
		Joins("INNER JOIN cards ON cards.id = card_attachments.card_id").
		Where("cards.board_id = ?", boardID).
		Find(&attachments).Error; err != nil {
		return err
	}
	for _, attachment := range attachments {
		_ = storage.DeleteAttachment(attachment.StoredName)
	}

	var cardIDs []uint
	if err := database.DB.Model(&models.Card{}).Where("board_id = ?", boardID).Pluck("id", &cardIDs).Error; err != nil {
		return err
	}

	if len(cardIDs) > 0 {
		if err := database.DB.Where("card_id IN ?", cardIDs).Delete(&models.CardSubscriber{}).Error; err != nil {
			return err
		}
		if err := database.DB.Exec("DELETE FROM card_tags WHERE card_id IN ?", cardIDs).Error; err != nil {
			return err
		}
		if err := database.DB.Where("card_id IN ?", cardIDs).Delete(&models.CardAttachment{}).Error; err != nil {
			return err
		}
		if err := database.DB.Where("card_id IN ?", cardIDs).Delete(&models.Comment{}).Error; err != nil {
			return err
		}
		if err := database.DB.Unscoped().Where("id IN ?", cardIDs).Delete(&models.Card{}).Error; err != nil {
			return err
		}
	}

	if err := database.DB.Where("board_id = ?", boardID).Delete(&models.Category{}).Error; err != nil {
		return err
	}
	if err := database.DB.Where("board_id = ?", boardID).Delete(&models.BoardTag{}).Error; err != nil {
		return err
	}
	if err := database.DB.Where("board_id = ?", boardID).Delete(&models.BoardMember{}).Error; err != nil {
		return err
	}
	return database.DB.Unscoped().Delete(&models.Board{}, boardID).Error
}
