package service

import (
	"context"
	"io"
	"time"

	"go.uber.org/zap"

	"lark-robot/internal/handler"
	"lark-robot/internal/larkbot"
	"lark-robot/internal/model"
	"lark-robot/internal/repository"
)

type MessageService struct {
	larkClient *larkbot.LarkClient
	logRepo    *repository.MessageLogRepo
	logger     *zap.Logger
}

func NewMessageService(larkClient *larkbot.LarkClient, logRepo *repository.MessageLogRepo, logger *zap.Logger) *MessageService {
	return &MessageService{
		larkClient: larkClient,
		logRepo:    logRepo,
		logger:     logger,
	}
}

// SendMessage sends a message and logs it.
func (s *MessageService) SendMessage(ctx context.Context, receiveID, receiveIDType, msgType, content, source string) (string, error) {
	msgID, err := s.larkClient.SendMessage(ctx, receiveID, receiveIDType, msgType, content)
	if err != nil {
		return "", err
	}

	// Determine chat type: non-chat_id types are always p2p
	chatType := ""
	if receiveIDType != "chat_id" {
		chatType = "p2p"
	} else {
		// For chat_id, look up from existing message logs
		chatType = s.logRepo.GetChatType(receiveID)
	}

	_ = s.logRepo.Create(&model.MessageLog{
		MessageID: msgID,
		ChatID:    receiveID,
		ChatType:  chatType,
		Direction: "out",
		MsgType:   msgType,
		Content:   content,
		Source:    source,
	})

	return msgID, nil
}

// ReplyMessage replies to a specific message and logs it.
func (s *MessageService) ReplyMessage(ctx context.Context, messageID, msgType, content string) (string, error) {
	replyMsgID, err := s.larkClient.ReplyMessage(ctx, messageID, msgType, content)
	if err != nil {
		return "", err
	}

	// Look up the original message's chat info for logging
	chatID := ""
	chatType := ""
	original, err := s.logRepo.GetByMessageID(messageID)
	if err == nil {
		chatID = original.ChatID
		chatType = original.ChatType
	}

	_ = s.logRepo.Create(&model.MessageLog{
		MessageID: replyMsgID,
		ChatID:    chatID,
		ChatType:  chatType,
		Direction: "out",
		MsgType:   msgType,
		Content:   content,
		Source:    "manual",
	})

	return replyMsgID, nil
}

// LogIncomingMessage logs a received message and its handler result.
func (s *MessageService) LogIncomingMessage(msg *handler.IncomingMessage, result *handler.Result, handlerName string) {
	_ = s.logRepo.Create(&model.MessageLog{
		MessageID:  msg.MessageID,
		ChatID:     msg.ChatID,
		ChatType:   msg.ChatType,
		SenderID:   msg.SenderID,
		SenderName: msg.SenderName,
		Direction:  "in",
		MsgType:    msg.MsgType,
		Content:    msg.Content,
		HandledBy:  handlerName,
		Source:     "event",
	})

	// Log the outgoing reply if one was produced
	if result != nil && result.Reply != nil {
		_ = s.logRepo.Create(&model.MessageLog{
			ChatID:    msg.ChatID,
			ChatType:  msg.ChatType,
			Direction: "out",
			MsgType:   result.Reply.MsgType,
			Content:   result.Reply.Content,
			HandledBy: handlerName,
			Source:    "event",
		})
	}
}

// GetLogs returns paginated message logs.
func (s *MessageService) GetLogs(q repository.MessageLogQuery) ([]model.MessageLog, int64, error) {
	return s.logRepo.List(q)
}

// ListConversations returns all distinct conversations from message logs.
// For p2p chats with missing sender names, it resolves them via the Lark API.
func (s *MessageService) ListConversations(ctx context.Context) ([]repository.Conversation, error) {
	conversations, err := s.logRepo.ListConversations()
	if err != nil {
		return nil, err
	}

	for i := range conversations {
		c := &conversations[i]
		if c.ChatType == "p2p" && c.SenderName == "" && c.SenderID != "" {
			userInfo, err := s.larkClient.GetUserInfo(ctx, c.SenderID)
			if err == nil && userInfo.Name != "" {
				c.SenderName = userInfo.Name
			}
		}
	}

	return conversations, nil
}

// DeleteMessage recalls a message via the Lark API and marks it as recalled in local logs.
func (s *MessageService) DeleteMessage(ctx context.Context, messageID string) error {
	if err := s.larkClient.DeleteMessage(ctx, messageID); err != nil {
		return err
	}
	_ = s.logRepo.RecallByMessageID(messageID)
	return nil
}

// GetMessageResource downloads a resource (image/file) from a Lark message.
func (s *MessageService) GetMessageResource(ctx context.Context, messageID, fileKey, resType string) (io.Reader, error) {
	return s.larkClient.GetMessageResource(ctx, messageID, fileKey, resType)
}

// CleanupGroupLogs deletes group chat logs older than the specified number of days.
func (s *MessageService) CleanupGroupLogs(days int) {
	before := time.Now().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
	count, err := s.logRepo.DeleteGroupLogsBefore(before)
	if err != nil {
		s.logger.Error("failed to cleanup group logs", zap.Error(err))
		return
	}
	if count > 0 {
		s.logger.Info("cleaned up group logs", zap.Int64("deleted", count), zap.Int("older_than_days", days))
	}
}

// CountToday returns today's message count.
func (s *MessageService) CountToday() (int64, error) {
	return s.logRepo.CountToday()
}
