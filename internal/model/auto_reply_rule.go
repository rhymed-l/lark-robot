package model

import (
	"time"

	"gorm.io/gorm"
)

type AutoReplyRule struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Keyword   string         `gorm:"size:255;not null;index" json:"keyword"`
	ReplyText string         `gorm:"type:text;not null" json:"reply_text"`
	MatchMode string         `gorm:"size:20;not null;default:contains" json:"match_mode"` // exact, contains, prefix
	ChatID    string         `gorm:"size:100;index" json:"chat_id"`                       // empty = all chats
	Enabled   bool           `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
