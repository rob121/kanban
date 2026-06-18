package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint    `gorm:"primaryKey"`
	Username     *string `gorm:"uniqueIndex;size:64"`
	Email        string  `gorm:"uniqueIndex;size:255;not null"`
	Name         string  `gorm:"size:255"`
	PasswordHash string  `gorm:"size:255"`
	AvatarURL    string  `gorm:"size:512"`
	Provider     string  `gorm:"size:64"`
	IsAdmin      bool   `gorm:"default:false"`
	Theme        string `gorm:"size:8;not null;default:'light'"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u User) UsernameString() string {
	if u.Username == nil {
		return ""
	}
	return *u.Username
}

type Board struct {
	ID          uint   `gorm:"primaryKey"`
	UserID      uint   `gorm:"index;not null"`
	Name        string `gorm:"size:255;not null"`
	Description string `gorm:"type:text"`
	Color       string `gorm:"size:32;not null;default:'#0d6efd'"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	User       User          `gorm:"foreignKey:UserID"`
	Categories []Category    `gorm:"foreignKey:BoardID"`
	Members    []BoardMember `gorm:"foreignKey:BoardID"`
	Tags       []BoardTag    `gorm:"foreignKey:BoardID"`
}

type BoardTag struct {
	ID        uint   `gorm:"primaryKey"`
	BoardID   uint   `gorm:"index;not null"`
	Name      string `gorm:"size:64;not null"`
	Color     string `gorm:"size:32;not null;default:'primary'"`
	Position  int    `gorm:"not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Board Board `gorm:"foreignKey:BoardID"`
}

// BoardMember grants a user access to a board with granular card permissions.
type BoardMember struct {
	ID        uint `gorm:"primaryKey"`
	BoardID   uint `gorm:"uniqueIndex:idx_board_user;not null"`
	UserID    uint `gorm:"uniqueIndex:idx_board_user;not null"`
	CanCreate bool `gorm:"not null;default:false"`
	CanUpdate bool `gorm:"not null;default:false"`
	CanDelete bool `gorm:"not null;default:false"`
	CanMove   bool `gorm:"not null;default:false"`
	CanAttach bool `gorm:"not null;default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Board Board `gorm:"foreignKey:BoardID"`
	User  User  `gorm:"foreignKey:UserID"`
}

type Category struct {
	ID        uint   `gorm:"primaryKey"`
	BoardID   uint   `gorm:"index;not null"`
	Name      string `gorm:"size:255;not null"`
	Position  int    `gorm:"not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Board Board  `gorm:"foreignKey:BoardID"`
	Cards []Card `gorm:"foreignKey:CategoryID"`
}

type Card struct {
	ID          uint       `gorm:"primaryKey"`
	BoardID     uint       `gorm:"index;not null"`
	CategoryID  uint       `gorm:"index;not null"`
	Title       string     `gorm:"size:255;not null"`
	Description string     `gorm:"type:text"`
	Priority    string     `gorm:"size:16;default:'medium'"`
	DueDate     *time.Time `gorm:"index"`
	AssigneeID  *uint      `gorm:"index"`
	CreatorID   *uint      `gorm:"index"`
	Position    int        `gorm:"not null;default:0"`
	Archived    bool       `gorm:"default:false"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	Board    Board     `gorm:"foreignKey:BoardID"`
	Category Category  `gorm:"foreignKey:CategoryID"`
	Assignee *User     `gorm:"foreignKey:AssigneeID"`
	Creator  *User     `gorm:"foreignKey:CreatorID"`
	Comments    []Comment        `gorm:"foreignKey:CardID"`
	Attachments []CardAttachment `gorm:"foreignKey:CardID"`
	Tags        []BoardTag       `gorm:"many2many:card_tags;"`
}

type CardAttachment struct {
	ID          uint   `gorm:"primaryKey"`
	CardID      uint   `gorm:"index;not null"`
	UserID      uint   `gorm:"index;not null"`
	Filename    string `gorm:"size:255;not null"`
	StoredName  string `gorm:"size:64;not null;uniqueIndex"`
	ContentType string `gorm:"size:128"`
	Size        int64  `gorm:"not null"`
	CreatedAt   time.Time

	Card Card `gorm:"foreignKey:CardID"`
	User User `gorm:"foreignKey:UserID"`
}

type Comment struct {
	ID        uint   `gorm:"primaryKey"`
	CardID    uint   `gorm:"index;not null"`
	UserID    uint   `gorm:"index;not null"`
	Body      string `gorm:"type:text;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Card Card `gorm:"foreignKey:CardID"`
	User User `gorm:"foreignKey:UserID"`
}

var DefaultCategories = []string{
	"Backlog",
	"Ready to Start",
	"In Progress",
	"Complete",
}

func SeedDefaultCategories(db *gorm.DB, boardID uint) error {
	for i, name := range DefaultCategories {
		cat := Category{
			BoardID:  boardID,
			Name:     name,
			Position: i,
		}
		if err := db.Create(&cat).Error; err != nil {
			return err
		}
	}
	return nil
}

var DefaultBoardTags = []struct {
	Name  string
	Color string
}{
	{"Feature", "primary"},
	{"Bug", "danger"},
	{"Urgent", "warning"},
	{"Docs", "info"},
}

func SeedDefaultTags(db *gorm.DB, boardID uint) error {
	for i, t := range DefaultBoardTags {
		tag := BoardTag{
			BoardID:  boardID,
			Name:     t.Name,
			Color:    t.Color,
			Position: i,
		}
		if err := db.Create(&tag).Error; err != nil {
			return err
		}
	}
	return nil
}
