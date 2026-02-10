package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"lark-robot/internal/larkbot"
	"lark-robot/internal/model"
	"lark-robot/internal/repository"
)

type ChatService struct {
	larkClient *larkbot.LarkClient
	repo       *repository.GroupRepo
	logger     *zap.Logger
}

func NewChatService(larkClient *larkbot.LarkClient, repo *repository.GroupRepo, logger *zap.Logger) *ChatService {
	return &ChatService{
		larkClient: larkClient,
		repo:       repo,
		logger:     logger,
	}
}

// SyncChats fetches all joined chats from Lark API and syncs to local database.
func (s *ChatService) SyncChats(ctx context.Context) ([]model.Group, error) {
	chats, err := s.larkClient.ListChats(ctx)
	if err != nil {
		return nil, err
	}

	var chatIDs []string
	for _, chat := range chats {
		group := &model.Group{
			ChatID:      chat.ChatID,
			Name:        chat.Name,
			Description: chat.Description,
			OwnerID:     chat.OwnerID,
			MemberCount: chat.MemberCount,
			External:    chat.External,
			SyncedAt:    time.Now(),
		}
		if err := s.repo.Upsert(group); err != nil {
			s.logger.Error("failed to upsert group", zap.String("chat_id", chat.ChatID), zap.Error(err))
			continue
		}
		chatIDs = append(chatIDs, chat.ChatID)
	}

	// Remove groups that the bot is no longer in
	if err := s.repo.DeleteNotIn(chatIDs); err != nil {
		s.logger.Error("failed to clean stale groups", zap.Error(err))
	}

	s.logger.Info("synced chats", zap.Int("count", len(chats)))
	groups, _, err := s.repo.List(1, 1000)
	return groups, err
}

// ListGroups returns cached groups from the database with pagination.
func (s *ChatService) ListGroups(page, pageSize int) ([]model.Group, int64, error) {
	return s.repo.List(page, pageSize)
}

// LeaveChat makes the bot leave a chat and removes it from local database.
func (s *ChatService) LeaveChat(ctx context.Context, chatID string) error {
	if err := s.larkClient.LeaveChat(ctx, chatID); err != nil {
		return err
	}
	return s.repo.DeleteByChatID(chatID)
}

// GroupCount returns the number of groups in the database.
func (s *ChatService) GroupCount() (int64, error) {
	return s.repo.Count()
}
