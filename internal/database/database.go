package database

import (
	"fmt"

	"github.com/rob121/kanban/internal/config"
	"github.com/rob121/kanban/internal/models"
	"github.com/rob121/kanban/internal/storage"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() error {
	cfg := config.C.Database
	var dialector gorm.Dialector

	switch cfg.Driver {
	case "mysql":
		dialector = mysql.Open(cfg.DSN)
	case "postgres", "postgresql":
		dialector = postgres.Open(cfg.DSN)
	default:
		dialector = sqlite.Open(cfg.DSN)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.APIToken{},
		&models.Board{},
		&models.BoardTag{},
		&models.BoardMember{},
		&models.Category{},
		&models.Card{},
		&models.Comment{},
		&models.CardAttachment{},
		&models.CardSubscriber{},
	); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	if err := backfillCardCreators(db); err != nil {
		return fmt.Errorf("backfill card creators: %w", err)
	}

	if err := storage.EnsureAttachmentsDir(); err != nil {
		return fmt.Errorf("attachments dir: %w", err)
	}

	DB = db
	return nil
}

func backfillCardCreators(db *gorm.DB) error {
	var cards []models.Card
	if err := db.Where("creator_id IS NULL").Find(&cards).Error; err != nil {
		return err
	}
	for _, card := range cards {
		var comment models.Comment
		if err := db.Where("card_id = ?", card.ID).Order("created_at asc").First(&comment).Error; err == nil {
			if err := db.Model(&card).Update("creator_id", comment.UserID).Error; err != nil {
				return err
			}
			continue
		}
		var board models.Board
		if err := db.Select("user_id").First(&board, card.BoardID).Error; err == nil {
			if err := db.Model(&card).Update("creator_id", board.UserID).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
