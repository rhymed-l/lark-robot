package model

import (
	"time"

	"gorm.io/gorm"
)

type ScheduledTask struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:255;not null" json:"name"`
	CronExpr  string         `gorm:"size:100;not null" json:"cron_expr"`
	ChatID    string         `gorm:"size:100;not null" json:"chat_id"`
	MsgType   string         `gorm:"size:20;not null;default:text" json:"msg_type"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	Enabled   bool           `gorm:"default:true" json:"enabled"`
	LastRunAt *time.Time     `json:"last_run_at"`
	NextRunAt *time.Time     `json:"next_run_at"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
