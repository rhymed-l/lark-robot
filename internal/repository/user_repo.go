package repository

import (
	"lark-robot/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

// Upsert creates or updates a user. On conflict, updates name/avatar/last_seen
// but preserves first_seen.
func (r *UserRepo) Upsert(user *model.User) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "open_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"union_id", "user_id", "name", "en_name", "avatar",
			"email", "job_title", "work_station", "employee_no",
			"gender", "leader_user_id", "join_time",
			"last_seen", "updated_at",
		}),
	}).Create(user).Error
}

// IncrementMsgCount atomically increments the message count for a user.
func (r *UserRepo) IncrementMsgCount(openID string) error {
	return r.db.Model(&model.User{}).
		Where("open_id = ?", openID).
		UpdateColumn("msg_count", gorm.Expr("msg_count + 1")).Error
}

// GetByOpenID returns a user by open_id.
func (r *UserRepo) GetByOpenID(openID string) (*model.User, error) {
	var user model.User
	err := r.db.Where("open_id = ?", openID).First(&user).Error
	return &user, err
}

// List returns paginated users with optional keyword search.
func (r *UserRepo) List(page, pageSize int, keyword string) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	tx := r.db.Model(&model.User{})
	if keyword != "" {
		tx = tx.Where("name LIKE ? OR en_name LIKE ? OR open_id LIKE ? OR employee_no LIKE ? OR email LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	tx.Count(&total)

	offset := (page - 1) * pageSize
	err := tx.Order("last_seen desc").Offset(offset).Limit(pageSize).Find(&users).Error
	return users, total, err
}

// Count returns the total number of users.
func (r *UserRepo) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Count(&count).Error
	return count, err
}
