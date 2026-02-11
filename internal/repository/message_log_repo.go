package repository

import (
	"lark-robot/internal/model"

	"gorm.io/gorm"
)

type MessageLogRepo struct {
	db *gorm.DB
}

func NewMessageLogRepo(db *gorm.DB) *MessageLogRepo {
	return &MessageLogRepo{db: db}
}

type MessageLogQuery struct {
	ChatID    string
	ChatType  string
	Direction string
	Source    string
	Page     int
	PageSize int
}

func (r *MessageLogRepo) List(q MessageLogQuery) ([]model.MessageLog, int64, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}

	tx := r.db.Model(&model.MessageLog{})
	if q.ChatID != "" {
		tx = tx.Where("chat_id = ?", q.ChatID)
	}
	if q.ChatType != "" {
		tx = tx.Where("chat_type = ?", q.ChatType)
	}
	if q.Direction != "" {
		tx = tx.Where("direction = ?", q.Direction)
	}
	if q.Source != "" {
		tx = tx.Where("source = ?", q.Source)
	}

	var total int64
	tx.Count(&total)

	var logs []model.MessageLog
	err := tx.Order("id desc").
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Find(&logs).Error

	return logs, total, err
}

func (r *MessageLogRepo) Create(log *model.MessageLog) error {
	return r.db.Create(log).Error
}

// Conversation represents a recent chat extracted from message logs.
type Conversation struct {
	ChatID      string `json:"chat_id"`
	ChatType    string `json:"chat_type"` // "p2p" or "group"
	SenderID    string `json:"sender_id"`
	SenderName  string `json:"sender_name"`
	LastContent string `json:"last_content"`
	LastTime    string `json:"last_time"`
	MsgCount    int64  `json:"msg_count"`
}

// ListConversations returns distinct chat_ids from message logs, ordered by most recent message.
func (r *MessageLogRepo) ListConversations() ([]Conversation, error) {
	var results []Conversation
	err := r.db.Model(&model.MessageLog{}).
		Select("chat_id, MAX(chat_type) as chat_type, MAX(sender_id) as sender_id, MAX(sender_name) as sender_name, MAX(content) as last_content, MAX(created_at) as last_time, COUNT(*) as msg_count").
		Group("chat_id").
		Order("last_time desc").
		Find(&results).Error
	return results, err
}

// DeleteGroupLogsBefore deletes group chat logs older than the given time. Private chats are not affected.
func (r *MessageLogRepo) DeleteGroupLogsBefore(before string) (int64, error) {
	result := r.db.Where("chat_type = 'group' AND created_at < ?", before).Delete(&model.MessageLog{})
	return result.RowsAffected, result.Error
}

// GetChatType returns the chat_type for a given chat_id from existing logs, or empty string if unknown.
func (r *MessageLogRepo) GetChatType(chatID string) string {
	var chatType string
	r.db.Model(&model.MessageLog{}).
		Select("chat_type").
		Where("chat_id = ? AND chat_type != ''", chatID).
		Limit(1).
		Scan(&chatType)
	return chatType
}

func (r *MessageLogRepo) CountToday() (int64, error) {
	var count int64
	err := r.db.Model(&model.MessageLog{}).
		Where("created_at >= date('now')").
		Count(&count).Error
	return count, err
}
