package repository

import (
	"lark-robot/internal/model"

	"gorm.io/gorm"
)

type ScheduledTaskRepo struct {
	db *gorm.DB
}

func NewScheduledTaskRepo(db *gorm.DB) *ScheduledTaskRepo {
	return &ScheduledTaskRepo{db: db}
}

func (r *ScheduledTaskRepo) List(page, pageSize int) ([]model.ScheduledTask, int64, error) {
	var tasks []model.ScheduledTask
	var total int64

	r.db.Model(&model.ScheduledTask{}).Count(&total)

	offset := (page - 1) * pageSize
	err := r.db.Order("id desc").Offset(offset).Limit(pageSize).Find(&tasks).Error
	return tasks, total, err
}

func (r *ScheduledTaskRepo) ListEnabled() ([]model.ScheduledTask, error) {
	var tasks []model.ScheduledTask
	err := r.db.Where("enabled = ?", true).Find(&tasks).Error
	return tasks, err
}

func (r *ScheduledTaskRepo) GetByID(id uint) (*model.ScheduledTask, error) {
	var task model.ScheduledTask
	err := r.db.First(&task, id).Error
	return &task, err
}

func (r *ScheduledTaskRepo) Create(task *model.ScheduledTask) error {
	return r.db.Create(task).Error
}

func (r *ScheduledTaskRepo) Update(task *model.ScheduledTask) error {
	return r.db.Save(task).Error
}

func (r *ScheduledTaskRepo) Delete(id uint) error {
	return r.db.Delete(&model.ScheduledTask{}, id).Error
}

func (r *ScheduledTaskRepo) ToggleEnabled(id uint) error {
	return r.db.Model(&model.ScheduledTask{}).
		Where("id = ?", id).
		Update("enabled", gorm.Expr("NOT enabled")).Error
}

func (r *ScheduledTaskRepo) UpdateLastRunAt(id uint) error {
	return r.db.Model(&model.ScheduledTask{}).
		Where("id = ?", id).
		Update("last_run_at", gorm.Expr("datetime('now')")).Error
}
