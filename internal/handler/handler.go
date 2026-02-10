package handler

import (
	"context"

	"go.uber.org/zap"
)

// IncomingMessage is a normalized representation of a received Lark message.
type IncomingMessage struct {
	MessageID   string
	ChatID      string
	ChatType    string // "p2p" or "group"
	SenderID    string
	SenderName  string
	MsgType     string // "text", "image", etc.
	Content     string // raw JSON content from Lark
	TextContent string // extracted plain text (for text messages)
	MentionBot  bool   // whether the bot was @mentioned
}

// Reply is what a handler wants to send back.
type Reply struct {
	MsgType string // "text", "interactive", etc.
	Content string // JSON content string
}

// Result is the outcome of a handler's processing.
type Result struct {
	Handled bool   // true = this handler claimed the message; stop chain
	Reply   *Reply // nil means no reply needed
}

// MessageHandler is the interface every handler must implement.
type MessageHandler interface {
	Name() string
	Handle(ctx context.Context, msg *IncomingMessage) (*Result, error)
}

// HandlerChain executes handlers in order until one claims the message.
type HandlerChain struct {
	handlers []MessageHandler
	logger   *zap.Logger
}

func NewHandlerChain(logger *zap.Logger, handlers ...MessageHandler) *HandlerChain {
	return &HandlerChain{
		handlers: handlers,
		logger:   logger,
	}
}

func (c *HandlerChain) SetHandlers(handlers []MessageHandler) {
	c.handlers = handlers
}

func (c *HandlerChain) Process(ctx context.Context, msg *IncomingMessage) (*Result, error) {
	for _, h := range c.handlers {
		result, err := h.Handle(ctx, msg)
		if err != nil {
			c.logger.Error("handler error",
				zap.String("handler", h.Name()),
				zap.Error(err),
			)
			continue
		}
		if result != nil && result.Handled {
			c.logger.Info("message handled",
				zap.String("handler", h.Name()),
				zap.String("message_id", msg.MessageID),
			)
			return result, nil
		}
	}
	return &Result{Handled: false}, nil
}
