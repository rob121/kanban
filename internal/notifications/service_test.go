package notifications

import (
	"testing"

	"github.com/rob121/kanban/internal/models"
)

func TestCommentRecipientIDs(t *testing.T) {
	ownerID := uint(1)
	assigneeID := uint(2)
	actorID := uint(3)

	card := models.Card{
		CreatorID:  &ownerID,
		AssigneeID: &assigneeID,
	}

	ids := commentRecipientIDs(card, actorID)
	if len(ids) != 2 {
		t.Fatalf("expected 2 recipients, got %d", len(ids))
	}
	if ids[0] != ownerID || ids[1] != assigneeID {
		t.Fatalf("unexpected recipient order: %v", ids)
	}

	same := models.Card{
		CreatorID:  &ownerID,
		AssigneeID: &ownerID,
	}
	ids = commentRecipientIDs(same, actorID)
	if len(ids) != 1 || ids[0] != ownerID {
		t.Fatalf("expected deduped owner only, got %v", ids)
	}

	self := models.Card{
		CreatorID:  &ownerID,
		AssigneeID: &assigneeID,
	}
	ids = commentRecipientIDs(self, ownerID)
	if len(ids) != 1 || ids[0] != assigneeID {
		t.Fatalf("owner commenting should only notify assignee, got %v", ids)
	}
}

func TestNotifyRecipientIDsWithSubscribers(t *testing.T) {
	ownerID := uint(1)
	subscriberID := uint(4)
	card := models.Card{
		ID:        10,
		CreatorID: &ownerID,
	}

	ids := notifyRecipientIDs(card, ownerID, []uint{subscriberID})
	if len(ids) != 1 || ids[0] != subscriberID {
		t.Fatalf("expected subscriber only when owner acts, got %v", ids)
	}
}
