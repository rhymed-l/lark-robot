package repository

import (
	"time"

	"lark-robot/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GroupRepo struct {
	db *gorm.DB
}

func NewGroupRepo(db *gorm.DB) *GroupRepo {
	return &GroupRepo{db: db}
}

func (r *GroupRepo) List(page, pageSize int) ([]model.Group, int64, error) {
	var groups []model.Group
	var total int64

	r.db.Model(&model.Group{}).Count(&total)

	offset := (page - 1) * pageSize
	err := r.db.Order("name asc").Offset(offset).Limit(pageSize).Find(&groups).Error
	return groups, total, err
}

func (r *GroupRepo) GetByChatID(chatID string) (*model.Group, error) {
	var group model.Group
	err := r.db.Where("chat_id = ?", chatID).First(&group).Error
	return &group, err
}

func (r *GroupRepo) Upsert(group *model.Group) error {
	group.SyncedAt = time.Now()
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "chat_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "description", "owner_id", "member_count", "external", "synced_at", "updated_at"}),
	}).Create(group).Error
}

func (r *GroupRepo) DeleteByChatID(chatID string) error {
	return r.db.Where("chat_id = ?", chatID).Delete(&model.Group{}).Error
}

func (r *GroupRepo) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.Group{}).Count(&count).Error
	return count, err
}

// DeleteNotIn removes groups whose chat_id is not in the given list.
func (r *GroupRepo) DeleteNotIn(chatIDs []string) error {
	if len(chatIDs) == 0 {
		return r.db.Where("1=1").Delete(&model.Group{}).Error
	}
	return r.db.Where("chat_id NOT IN ?", chatIDs).Delete(&model.Group{}).Error
}
