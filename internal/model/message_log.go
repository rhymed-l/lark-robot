package model

import "time"

type MessageLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	MessageID string    `gorm:"size:100;index" json:"message_id"`
	ChatID    string    `gorm:"size:100;index" json:"chat_id"`
	ChatType  string    `gorm:"size:10" json:"chat_type"` // "p2p" or "group"
	SenderID   string    `gorm:"size:100" json:"sender_id"`
	SenderName string    `gorm:"size:100" json:"sender_name"`
	Direction string    `gorm:"size:10;not null" json:"direction"` // "in" or "out"
	MsgType   string    `gorm:"size:20" json:"msg_type"`
	Content   string    `gorm:"type:text" json:"content"`
	HandledBy string    `gorm:"size:50" json:"handled_by"`
	Source    string    `gorm:"size:20" json:"source"` // "event", "scheduled", "manual"
	Recalled bool      `gorm:"default:false" json:"recalled"`
	CreatedAt time.Time `json:"created_at"`
}
