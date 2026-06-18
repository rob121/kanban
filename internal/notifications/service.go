package notifications

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log"
	"strings"

	"github.com/rob121/kanban/internal/config"
	"github.com/rob121/kanban/internal/database"
	"github.com/rob121/kanban/internal/models"
	"github.com/rob121/kanban/internal/subscriptions"
	"github.com/rob121/kanban/mailer"
)

//go:embed templates/*.html
var templateFS embed.FS

var defaultService *Service

// Service dispatches notifications through configured channels (email today).
type Service struct {
	email *emailChannel
}

// Init prepares the notification service. Call after config and mailer are loaded.
func Init() {
	defaultService = &Service{
		email: newEmailChannel(),
	}
}

// CommentAdded notifies the card owner, assignee, and subscribers when a comment is posted.
func CommentAdded(cardID, commentID, actorID uint) {
	if defaultService == nil {
		return
	}
	go defaultService.commentAdded(cardID, commentID, actorID)
}

// CardMoved notifies interested users when the card changes columns.
func CardMoved(cardID, actorID, oldCategoryID uint) {
	if defaultService == nil {
		return
	}
	go defaultService.cardMoved(cardID, actorID, oldCategoryID)
}

// CardUpdated notifies interested users when card details change.
func CardUpdated(cardID, actorID uint) {
	if defaultService == nil {
		return
	}
	go defaultService.cardUpdated(cardID, actorID)
}

type cardContext struct {
	Card        models.Card
	Board       models.Board
	Category    models.Category
	OldCategory *models.Category
}

func (s *Service) commentAdded(cardID, commentID, actorID uint) {
	ctx, comment, actor, err := loadCommentContext(cardID, commentID, actorID)
	if err != nil {
		log.Printf("notifications: comment: %v", err)
		return
	}

	recipients := notifyUsers(ctx.Card, actorID)
	if len(recipients) == 0 {
		return
	}

	cardURL := cardLink(ctx.Card.BoardID, ctx.Card.ID)
	subject := fmt.Sprintf("%s — New comment on %s", config.C.Branding.AppName, ctx.Card.Title)
	preheader := fmt.Sprintf("%s commented on %s", actor.Name, ctx.Card.Title)

	for _, user := range recipients {
		body, err := renderNotificationBody("comment.html", map[string]any{
			"RecipientName": user.Name,
			"ActorName":     actor.Name,
			"CardTitle":     ctx.Card.Title,
			"BoardName":     ctx.Board.Name,
			"ColumnName":    ctx.Category.Name,
			"Priority":      priorityLabel(ctx.Card.Priority),
			"CommentBody":   comment.Body,
			"CardURL":       cardURL,
			"BrandColor":    config.C.Branding.BrandColor,
		})
		if err != nil {
			log.Printf("notifications: comment template: %v", err)
			return
		}
		s.email.send(user, subject, preheader, body)
	}
}

func (s *Service) cardMoved(cardID, actorID, oldCategoryID uint) {
	ctx, actor, err := loadCardContextWithActor(cardID, oldCategoryID, actorID)
	if err != nil {
		log.Printf("notifications: move: %v", err)
		return
	}

	recipients := notifyUsers(ctx.Card, actorID)
	if len(recipients) == 0 {
		return
	}

	oldName := "Previous column"
	if ctx.OldCategory != nil {
		oldName = ctx.OldCategory.Name
	}

	cardURL := cardLink(ctx.Card.BoardID, ctx.Card.ID)
	subject := fmt.Sprintf("%s — Card moved on %s", config.C.Branding.AppName, ctx.Board.Name)
	preheader := fmt.Sprintf("%s moved %s to %s", actor.Name, ctx.Card.Title, ctx.Category.Name)

	for _, user := range recipients {
		body, err := renderNotificationBody("card_moved.html", map[string]any{
			"RecipientName": user.Name,
			"ActorName":     actor.Name,
			"CardTitle":     ctx.Card.Title,
			"BoardName":     ctx.Board.Name,
			"FromColumn":    oldName,
			"ToColumn":      ctx.Category.Name,
			"Priority":      priorityLabel(ctx.Card.Priority),
			"CardURL":       cardURL,
			"BrandColor":    config.C.Branding.BrandColor,
		})
		if err != nil {
			log.Printf("notifications: move template: %v", err)
			return
		}
		s.email.send(user, subject, preheader, body)
	}
}

func (s *Service) cardUpdated(cardID, actorID uint) {
	ctx, actor, err := loadCardContextWithActor(cardID, 0, actorID)
	if err != nil {
		log.Printf("notifications: update: %v", err)
		return
	}

	recipients := notifyUsers(ctx.Card, actorID)
	if len(recipients) == 0 {
		return
	}

	cardURL := cardLink(ctx.Card.BoardID, ctx.Card.ID)
	subject := fmt.Sprintf("%s — Card updated on %s", config.C.Branding.AppName, ctx.Board.Name)
	preheader := fmt.Sprintf("%s updated %s", actor.Name, ctx.Card.Title)

	for _, user := range recipients {
		body, err := renderNotificationBody("card_updated.html", map[string]any{
			"RecipientName": user.Name,
			"ActorName":     actor.Name,
			"CardTitle":     ctx.Card.Title,
			"BoardName":     ctx.Board.Name,
			"ColumnName":    ctx.Category.Name,
			"Priority":      priorityLabel(ctx.Card.Priority),
			"CardURL":       cardURL,
			"BrandColor":    config.C.Branding.BrandColor,
		})
		if err != nil {
			log.Printf("notifications: update template: %v", err)
			return
		}
		s.email.send(user, subject, preheader, body)
	}
}

func notifyUsers(card models.Card, actorID uint) []models.User {
	var users []models.User
	for _, id := range notifyRecipientIDsForCard(card, actorID) {
		var user models.User
		if err := database.DB.First(&user, id).Error; err != nil {
			continue
		}
		if strings.TrimSpace(user.Email) == "" {
			continue
		}
		users = append(users, user)
	}
	return users
}

func notifyRecipientIDsForCard(card models.Card, actorID uint) []uint {
	return notifyRecipientIDs(card, actorID, subscriptions.SubscriberIDs(card.ID))
}

func notifyRecipientIDs(card models.Card, actorID uint, subscriberIDs []uint) []uint {
	seen := map[uint]bool{actorID: true}
	var ids []uint

	add := func(id uint) {
		if id == 0 || seen[id] {
			return
		}
		seen[id] = true
		ids = append(ids, id)
	}

	add(card.OwnerID())
	if card.AssigneeID != nil {
		add(*card.AssigneeID)
	}
	for _, subID := range subscriberIDs {
		add(subID)
	}
	return ids
}

// commentRecipientIDs supports tests without a database.
func commentRecipientIDs(card models.Card, actorID uint) []uint {
	return notifyRecipientIDs(card, actorID, nil)
}

func loadCommentContext(cardID, commentID, actorID uint) (cardContext, models.Comment, models.User, error) {
	ctx, actor, err := loadCardContextWithActor(cardID, 0, actorID)
	if err != nil {
		return cardContext{}, models.Comment{}, models.User{}, err
	}

	var comment models.Comment
	if err := database.DB.Preload("User").First(&comment, commentID).Error; err != nil {
		return cardContext{}, models.Comment{}, models.User{}, err
	}

	return ctx, comment, actor, nil
}

func loadCardContext(cardID, oldCategoryID uint) (cardContext, error) {
	var card models.Card
	if err := database.DB.Preload("Assignee").Preload("Creator").Preload("Category").First(&card, cardID).Error; err != nil {
		return cardContext{}, err
	}

	var board models.Board
	if err := database.DB.First(&board, card.BoardID).Error; err != nil {
		return cardContext{}, err
	}

	ctx := cardContext{
		Card:     card,
		Board:    board,
		Category: card.Category,
	}

	if oldCategoryID != 0 && oldCategoryID != card.CategoryID {
		var oldCat models.Category
		if err := database.DB.First(&oldCat, oldCategoryID).Error; err == nil {
			ctx.OldCategory = &oldCat
		}
	}

	return ctx, nil
}

func loadCardContextWithActor(cardID, oldCategoryID, actorID uint) (cardContext, models.User, error) {
	ctx, err := loadCardContext(cardID, oldCategoryID)
	if err != nil {
		return cardContext{}, models.User{}, err
	}
	var actor models.User
	if err := database.DB.First(&actor, actorID).Error; err != nil {
		return cardContext{}, models.User{}, err
	}
	return ctx, actor, nil
}

func cardLink(boardID, cardID uint) string {
	base := strings.TrimRight(config.C.BaseURL, "/")
	return fmt.Sprintf("%s/boards/%d?card=%d", base, boardID, cardID)
}

func priorityLabel(priority string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "high":
		return "High"
	case "low":
		return "Low"
	default:
		return "Medium"
	}
}

func renderNotificationBody(name string, data map[string]any) (template.HTML, error) {
	raw, err := templateFS.ReadFile("templates/" + name)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New(name).Parse(string(raw))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

type emailChannel struct{}

func newEmailChannel() *emailChannel {
	return &emailChannel{}
}

func (c *emailChannel) send(user models.User, subject, preheader string, body template.HTML) {
	if !mailer.Enabled() {
		return
	}
	email := strings.TrimSpace(user.Email)
	if email == "" {
		return
	}

	err := mailer.New().
		To(email).
		Add("Preheader", preheader).
		Subject(subject).
		Body(body).
		Send()
	if err != nil {
		log.Printf("notifications: email to %s: %v", email, err)
	}
}
