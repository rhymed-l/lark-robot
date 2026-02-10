package service

import (
	"go.uber.org/zap"

	"lark-robot/internal/handler"
	"lark-robot/internal/model"
	"lark-robot/internal/repository"
)

type ReplyService struct {
	repo           *repository.AutoReplyRuleRepo
	keywordHandler *handler.KeywordHandler
	logger         *zap.Logger
}

func NewReplyService(repo *repository.AutoReplyRuleRepo, keywordHandler *handler.KeywordHandler, logger *zap.Logger) *ReplyService {
	return &ReplyService{
		repo:           repo,
		keywordHandler: keywordHandler,
		logger:         logger,
	}
}

// ReloadRules loads enabled rules from the database and updates the KeywordHandler.
func (s *ReplyService) ReloadRules() error {
	rules, err := s.repo.ListEnabled()
	if err != nil {
		return err
	}
	keywordRules := toKeywordRules(rules)
	s.keywordHandler.UpdateRules(keywordRules)
	s.logger.Info("reloaded auto-reply rules", zap.Int("count", len(keywordRules)))
	return nil
}

func (s *ReplyService) List(page, pageSize int) ([]model.AutoReplyRule, int64, error) {
	return s.repo.List(page, pageSize)
}

func (s *ReplyService) GetByID(id uint) (*model.AutoReplyRule, error) {
	return s.repo.GetByID(id)
}

func (s *ReplyService) Create(rule *model.AutoReplyRule) error {
	if err := s.repo.Create(rule); err != nil {
		return err
	}
	return s.ReloadRules()
}

func (s *ReplyService) Update(rule *model.AutoReplyRule) error {
	if err := s.repo.Update(rule); err != nil {
		return err
	}
	return s.ReloadRules()
}

func (s *ReplyService) Delete(id uint) error {
	if err := s.repo.Delete(id); err != nil {
		return err
	}
	return s.ReloadRules()
}

func (s *ReplyService) Toggle(id uint) error {
	if err := s.repo.ToggleEnabled(id); err != nil {
		return err
	}
	return s.ReloadRules()
}

func toKeywordRules(rules []model.AutoReplyRule) []handler.KeywordRule {
	result := make([]handler.KeywordRule, len(rules))
	for i, r := range rules {
		result[i] = handler.KeywordRule{
			ID:        r.ID,
			Keyword:   r.Keyword,
			ReplyText: r.ReplyText,
			MatchMode: r.MatchMode,
			ChatID:    r.ChatID,
			Enabled:   r.Enabled,
		}
	}
	return result
}
