package subscriptions

import (
	"testing"

	"github.com/rob121/kanban/internal/models"
)

func TestCanSubscribe(t *testing.T) {
	ownerID := uint(1)
	assigneeID := uint(2)
	viewerID := uint(3)

	card := models.Card{
		CreatorID:  &ownerID,
		AssigneeID: &assigneeID,
	}

	if !CanSubscribe(card, viewerID) {
		t.Fatal("viewer should be able to subscribe")
	}
	if CanSubscribe(card, ownerID) {
		t.Fatal("owner should not subscribe")
	}
	if CanSubscribe(card, assigneeID) {
		t.Fatal("assignee should not subscribe")
	}
}
