package subscriptions

import (
	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
)

// CanSubscribe reports whether a viewer may opt into card notifications.
func CanSubscribe(card models.Card, userID uint) bool {
	if card.IsOwnedBy(userID) {
		return false
	}
	if card.AssigneeID != nil && *card.AssigneeID == userID {
		return false
	}
	return true
}

func IsSubscribed(cardID, userID uint) bool {
	var count int64
	database.DB.Model(&models.CardSubscriber{}).
		Where("card_id = ? AND user_id = ?", cardID, userID).
		Count(&count)
	return count > 0
}

func Subscribe(cardID, userID uint) error {
	if IsSubscribed(cardID, userID) {
		return nil
	}
	return database.DB.Create(&models.CardSubscriber{
		CardID: cardID,
		UserID: userID,
	}).Error
}

func Unsubscribe(cardID, userID uint) error {
	return database.DB.Where("card_id = ? AND user_id = ?", cardID, userID).
		Delete(&models.CardSubscriber{}).Error
}

func SubscriberIDs(cardID uint) []uint {
	var subs []models.CardSubscriber
	database.DB.Where("card_id = ?", cardID).Find(&subs)
	ids := make([]uint, len(subs))
	for i, s := range subs {
		ids[i] = s.UserID
	}
	return ids
}

// ClearForUser removes subscriptions when the user becomes owner or assignee.
func ClearForUser(cardID, userID uint) {
	database.DB.Where("card_id = ? AND user_id = ?", cardID, userID).
		Delete(&models.CardSubscriber{})
}
