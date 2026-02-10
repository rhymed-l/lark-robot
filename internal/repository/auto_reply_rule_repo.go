package repository

import (
	"lark-robot/internal/model"

	"gorm.io/gorm"
)

type AutoReplyRuleRepo struct {
	db *gorm.DB
}

func NewAutoReplyRuleRepo(db *gorm.DB) *AutoReplyRuleRepo {
	return &AutoReplyRuleRepo{db: db}
}

func (r *AutoReplyRuleRepo) List(page, pageSize int) ([]model.AutoReplyRule, int64, error) {
	var rules []model.AutoReplyRule
	var total int64

	r.db.Model(&model.AutoReplyRule{}).Count(&total)

	offset := (page - 1) * pageSize
	err := r.db.Order("id desc").Offset(offset).Limit(pageSize).Find(&rules).Error
	return rules, total, err
}

func (r *AutoReplyRuleRepo) ListEnabled() ([]model.AutoReplyRule, error) {
	var rules []model.AutoReplyRule
	err := r.db.Where("enabled = ?", true).Find(&rules).Error
	return rules, err
}

func (r *AutoReplyRuleRepo) GetByID(id uint) (*model.AutoReplyRule, error) {
	var rule model.AutoReplyRule
	err := r.db.First(&rule, id).Error
	return &rule, err
}

func (r *AutoReplyRuleRepo) Create(rule *model.AutoReplyRule) error {
	return r.db.Create(rule).Error
}

func (r *AutoReplyRuleRepo) Update(rule *model.AutoReplyRule) error {
	return r.db.Save(rule).Error
}

func (r *AutoReplyRuleRepo) Delete(id uint) error {
	return r.db.Delete(&model.AutoReplyRule{}, id).Error
}

func (r *AutoReplyRuleRepo) ToggleEnabled(id uint) error {
	return r.db.Model(&model.AutoReplyRule{}).
		Where("id = ?", id).
		Update("enabled", gorm.Expr("NOT enabled")).Error
}
