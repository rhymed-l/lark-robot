package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"lark-robot/config"
	"lark-robot/internal/broadcast"
	"lark-robot/internal/database"
	"lark-robot/internal/handler"
	"lark-robot/internal/larkbot"
	"lark-robot/internal/repository"
	"lark-robot/internal/scheduler"
	"lark-robot/internal/server"
	"lark-robot/internal/service"
	"lark-robot/static"
)

type App struct {
	config         *config.Config
	logger         *zap.Logger
	larkClient     *larkbot.LarkClient
	wsClient       *larkws.Client
	handlerChain   *handler.HandlerChain
	keywordHandler *handler.KeywordHandler
	sched          *scheduler.Scheduler
	router         *server.Router
	httpServer     *http.Server

	Broadcaster      *broadcast.MessageBroadcaster
	messageService   *service.MessageService
	replyService     *service.ReplyService
	chatService      *service.ChatService
	schedulerService *service.SchedulerService
}

func New(cfg *config.Config) (*App, error) {
	// 1. Initialize logger
	logger, err := initLogger(cfg.Log)
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}

	// 2. Initialize database
	db, err := database.Init(cfg.Database.Path, logger)
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	// 3. Create repositories
	ruleRepo := repository.NewAutoReplyRuleRepo(db)
	taskRepo := repository.NewScheduledTaskRepo(db)
	logRepo := repository.NewMessageLogRepo(db)
	groupRepo := repository.NewGroupRepo(db)

	// 4. Create Lark client
	larkClient := larkbot.NewLarkClient(cfg.Lark.AppID, cfg.Lark.AppSecret, cfg.Lark.BaseURL)

	// 5. Create message service
	msgService := service.NewMessageService(larkClient, logRepo, logger)

	// 6. Build handler chain
	keywordHandler := handler.NewKeywordHandler(nil)
	handlerChain := handler.NewHandlerChain(logger, keywordHandler, handler.NewDefaultHandler())

	// 7. Create services
	replyService := service.NewReplyService(ruleRepo, keywordHandler, logger)
	chatService := service.NewChatService(larkClient, groupRepo, logger)

	if err := replyService.ReloadRules(); err != nil {
		logger.Warn("failed to load auto-reply rules", zap.Error(err))
	}

	// 8. Create scheduler
	scheduledSendFunc := func(ctx context.Context, chatID, msgType, content, source string) (string, error) {
		return msgService.SendMessage(ctx, chatID, "chat_id", msgType, content, source)
	}
	sched := scheduler.New(
		scheduledSendFunc,
		taskRepo.UpdateLastRunAt,
		logger,
	)
	schedulerService := service.NewSchedulerService(taskRepo, sched, logger)

	// 9. Create message broadcaster for SSE
	broadcaster := broadcast.NewMessageBroadcaster()

	// 10. Set up Lark event dispatcher (WebSocket long connection)
	eventDispatcher := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			msg := parseIncomingMessage(event)

			// Resolve sender name
			userInfo, err := larkClient.GetUserInfo(ctx, msg.SenderID)
			if err != nil {
				logger.Debug("failed to get user info", zap.String("sender_id", msg.SenderID), zap.Error(err))
			}
			senderName := msg.SenderID
			if userInfo != nil && userInfo.Name != "" {
				senderName = userInfo.Name
				msg.SenderName = userInfo.Name
			}

			logger.Info("received message",
				zap.String("chat_id", msg.ChatID),
				zap.String("sender", senderName),
				zap.String("text", msg.TextContent),
			)

			// Broadcast incoming message to SSE subscribers
			broadcaster.Publish(broadcast.MessageEvent{
				ID:         msg.MessageID,
				ChatID:     msg.ChatID,
				ChatType:   msg.ChatType,
				SenderID:   msg.SenderID,
				SenderName: senderName,
				Direction:  "in",
				MsgType:    msg.MsgType,
				Content:    msg.Content,
				CreatedAt:  time.Now(),
			})

			result, err := handlerChain.Process(ctx, msg)
			if err != nil {
				logger.Error("handler chain error", zap.Error(err))
				return nil
			}

			if result.Handled && result.Reply != nil {
				_, sendErr := larkClient.ReplyMessage(ctx, msg.MessageID, result.Reply.MsgType, result.Reply.Content)
				if sendErr != nil {
					logger.Error("failed to send reply", zap.Error(sendErr))
				}
				// Broadcast auto-reply to SSE subscribers
				broadcaster.Publish(broadcast.MessageEvent{
					ChatID:    msg.ChatID,
					ChatType:  msg.ChatType,
					Direction: "out",
					MsgType:   result.Reply.MsgType,
					Content:   result.Reply.Content,
					CreatedAt: time.Now(),
				})
			}

			msgService.LogIncomingMessage(msg, result, "")
			return nil
		})

	// Build WebSocket client for long connection
	wsClient := larkws.NewClient(cfg.Lark.AppID, cfg.Lark.AppSecret,
		larkws.WithEventHandler(eventDispatcher),
		larkws.WithLogLevel(larkcore.LogLevelInfo),
	)

	// 10. Create router with embedded frontend
	distFS := static.DistFS()
	frontendFS := server.TryLoadFrontendFS(distFS)

	router := server.NewRouter(server.RouterConfig{
		Mode:             cfg.Server.Mode,
		Logger:           logger,
		AuthUsername:      cfg.Auth.Username,
		AuthPassword:      cfg.Auth.Password,
		AuthSecret:        cfg.Auth.Secret,
		ChatService:      chatService,
		MessageService:   msgService,
		SchedulerService: schedulerService,
		ReplyService:     replyService,
		Broadcaster:      broadcaster,
		FrontendFS:       frontendFS,
		EmbeddedFS:       distFS,
	})

	return &App{
		config:           cfg,
		logger:           logger,
		larkClient:       larkClient,
		wsClient:         wsClient,
		handlerChain:     handlerChain,
		keywordHandler:   keywordHandler,
		sched:            sched,
		Broadcaster:      broadcaster,
		router:           router,
		messageService:   msgService,
		replyService:     replyService,
		chatService:      chatService,
		schedulerService: schedulerService,
	}, nil
}

func (a *App) Start() error {
	// Load and start scheduled tasks
	if err := a.schedulerService.LoadAndStartAll(); err != nil {
		a.logger.Warn("failed to load scheduled tasks", zap.Error(err))
	}
	a.sched.Start()

	// Register daily cleanup: delete group chat logs older than 7 days (runs at 02:00 every day)
	a.sched.AddCleanupJob("0 0 2 * * *", func() {
		a.messageService.CleanupGroupLogs(7)
	})

	// Start Lark WebSocket long connection in background
	go func() {
		a.logger.Info("starting lark websocket connection")
		if err := a.wsClient.Start(context.Background()); err != nil {
			a.logger.Error("lark websocket connection error", zap.Error(err))
		}
	}()

	// Start HTTP server for admin dashboard
	addr := fmt.Sprintf(":%d", a.config.Server.Port)
	a.httpServer = &http.Server{
		Addr:    addr,
		Handler: a.router.Engine,
	}

	go func() {
		a.logger.Info("admin dashboard started", zap.String("addr", addr))
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Fatal("server error", zap.Error(err))
		}
	}()

	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	a.sched.Stop()
	if a.httpServer != nil {
		if err := a.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("http server shutdown: %w", err)
		}
	}
	a.logger.Info("application shutdown complete")
	return nil
}

func parseIncomingMessage(event *larkim.P2MessageReceiveV1) *handler.IncomingMessage {
	msg := event.Event.Message
	senderID := ""
	if event.Event.Sender != nil && event.Event.Sender.SenderId != nil {
		senderID = deref(event.Event.Sender.SenderId.OpenId)
	}

	content := deref(msg.Content)
	textContent := extractText(content)

	return &handler.IncomingMessage{
		MessageID:   deref(msg.MessageId),
		ChatID:      deref(msg.ChatId),
		ChatType:    deref(msg.ChatType),
		SenderID:    senderID,
		MsgType:     deref(msg.MessageType),
		Content:     content,
		TextContent: textContent,
		MentionBot:  containsBotMention(msg.Mentions),
	}
}

func extractText(content string) string {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(content), &m); err != nil {
		return content
	}
	if text, ok := m["text"].(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func containsBotMention(mentions []*larkim.MentionEvent) bool {
	for _, m := range mentions {
		if m.Key != nil && *m.Key == "at_all" {
			continue
		}
		if m.Id != nil && m.Id.OpenId != nil {
			return true
		}
	}
	return false
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func initLogger(cfg config.LogConfig) (*zap.Logger, error) {
	level := zapcore.InfoLevel
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = zapcore.DebugLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}

	var cores []zapcore.Core

	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level))

	if cfg.File != "" {
		file, err := os.OpenFile(cfg.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		jsonEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
		cores = append(cores, zapcore.NewCore(jsonEncoder, zapcore.AddSync(file), level))
	}

	core := zapcore.NewTee(cores...)
	return zap.New(core), nil
}
