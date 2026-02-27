package model

import "time"

type Group struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ChatID      string    `gorm:"size:100;uniqueIndex;not null" json:"chat_id"`
	Name        string    `gorm:"size:255" json:"name"`
	Avatar      string    `gorm:"size:500" json:"avatar"`
	Description string    `gorm:"type:text" json:"description"`
	ChatMode    string    `gorm:"size:20" json:"chat_mode"`
	ChatType    string    `gorm:"size:20" json:"chat_type"`
	ChatTag     string    `gorm:"size:50" json:"chat_tag"`
	OwnerID     string    `gorm:"size:100" json:"owner_id"`
	MemberCount int       `json:"member_count"`
	BotCount    int       `json:"bot_count"`
	External    bool      `json:"external"`
	SyncedAt    time.Time `json:"synced_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
