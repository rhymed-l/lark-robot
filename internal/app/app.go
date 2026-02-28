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
	userService      *service.UserService
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
	userRepo := repository.NewUserRepo(db)

	// 4. Create Lark client and fetch bot info
	larkClient := larkbot.NewLarkClient(cfg.Lark.AppID, cfg.Lark.AppSecret, cfg.Lark.BaseURL)
	if err := larkClient.FetchBotInfo(context.Background()); err != nil {
		logger.Warn("failed to fetch bot info", zap.Error(err))
	} else {
		logger.Info("bot info loaded", zap.String("open_id", larkClient.BotOpenID), zap.String("name", larkClient.BotName))
	}

	// 5. Create services
	msgService := service.NewMessageService(larkClient, logRepo, logger)
	userService := service.NewUserService(larkClient, userRepo, logger)

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
		taskRepo.UpdateNextRunAt,
		logger,
	)
	schedulerService := service.NewSchedulerService(taskRepo, sched, logger)

	// 9. Create message broadcaster for SSE
	broadcaster := broadcast.NewMessageBroadcaster()

	// 10. Set up Lark event dispatcher (WebSocket long connection)
	eventDispatcher := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			msg := parseIncomingMessage(event, larkClient.BotOpenID)

			// Auto-sync group if not yet in DB
			if msg.ChatType == "group" {
				go chatService.AutoSyncGroup(ctx, msg.ChatID)
			}

			// Resolve sender name via UserService (cache -> DB -> Lark API)
			userInfo, err := userService.GetUserInfo(ctx, msg.SenderID)
			if err != nil {
				logger.Debug("failed to get user info", zap.String("sender_id", msg.SenderID), zap.Error(err))
			}
			senderName := msg.SenderID
			if userInfo != nil && userInfo.Name != "" {
				senderName = userInfo.Name
				msg.SenderName = userInfo.Name
			}

			// Persist user info and increment message count asynchronously
			go userService.OnMessageReceived(ctx, msg.SenderID)

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
		}).
		OnP2MessageRecalledV1(func(ctx context.Context, event *larkim.P2MessageRecalledV1) error {
			if event.Event == nil || event.Event.MessageId == nil {
				return nil
			}
			messageID := *event.Event.MessageId
			chatID := ""
			if event.Event.ChatId != nil {
				chatID = *event.Event.ChatId
			}

			logger.Info("message recalled",
				zap.String("message_id", messageID),
				zap.String("chat_id", chatID),
			)

			// Mark as recalled in database
			_ = logRepo.RecallByMessageID(messageID)

			// Broadcast recall event to SSE subscribers
			broadcaster.Publish(broadcast.MessageEvent{
				ChatID:    chatID,
				Recalled:  true,
				MessageID: messageID,
				CreatedAt: time.Now(),
			})

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
		LarkClient:       larkClient,
		ChatService:      chatService,
		MessageService:   msgService,
		SchedulerService: schedulerService,
		ReplyService:     replyService,
		UserService:      userService,
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
		userService:      userService,
	}, nil
}

func (a *App) Start() error {
	// Load and start scheduled tasks
	if err := a.schedulerService.LoadAndStartAll(); err != nil {
		a.logger.Warn("failed to load scheduled tasks", zap.Error(err))
	}
	a.sched.Start()

	// Register daily cleanup: delete group chat logs older than 7 days (runs at 02:00 every day)
	if err := a.sched.AddCleanupJob("0 0 2 * * *", func() {
		a.messageService.CleanupGroupLogs(7)
	}); err != nil {
		a.logger.Error("failed to register cleanup job", zap.Error(err))
	}

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

func parseIncomingMessage(event *larkim.P2MessageReceiveV1, botOpenID string) *handler.IncomingMessage {
	msg := event.Event.Message
	senderID := ""
	if event.Event.Sender != nil && event.Event.Sender.SenderId != nil {
		senderID = deref(event.Event.Sender.SenderId.OpenId)
	}

	content := deref(msg.Content)

	// Replace @mention placeholders with real names in both content and text
	content = replaceMentions(content, msg.Mentions)
	textContent := extractText(content)

	return &handler.IncomingMessage{
		MessageID:   deref(msg.MessageId),
		ChatID:      deref(msg.ChatId),
		ChatType:    deref(msg.ChatType),
		SenderID:    senderID,
		MsgType:     deref(msg.MessageType),
		Content:     content,
		TextContent: textContent,
		MentionBot:  containsBotMention(msg.Mentions, botOpenID),
	}
}

func extractText(content string) string {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(content), &m); err != nil {
		return content
	}
	// Plain text message: {"text":"hello"}
	if text, ok := m["text"].(string); ok {
		return strings.TrimSpace(text)
	}
	// Rich text (post) message: {"title":"","content":[[{"tag":"text","text":"hello"},...]]}
	if contentArr, ok := m["content"].([]interface{}); ok {
		return extractPostText(contentArr)
	}
	return ""
}

// extractPostText extracts plain text from a post message's content array.
func extractPostText(content []interface{}) string {
	var sb strings.Builder
	for _, line := range content {
		lineArr, ok := line.([]interface{})
		if !ok {
			continue
		}
		for _, elem := range lineArr {
			elemMap, ok := elem.(map[string]interface{})
			if !ok {
				continue
			}
			tag, _ := elemMap["tag"].(string)
			switch tag {
			case "text":
				if text, ok := elemMap["text"].(string); ok {
					sb.WriteString(text)
				}
			case "at":
				if name, ok := elemMap["user_name"].(string); ok && name != "" {
					sb.WriteString("@[" + name + "]")
				}
			}
		}
	}
	return strings.TrimSpace(sb.String())
}

// replaceMentions replaces @_user_1 placeholders with actual names from the mentions list.
// Names are wrapped as @[Name] so the frontend can identify the full mention boundary.
func replaceMentions(text string, mentions []*larkim.MentionEvent) string {
	for _, m := range mentions {
		if m.Key != nil && m.Name != nil {
			text = strings.ReplaceAll(text, *m.Key, "@["+*m.Name+"]")
		}
	}
	// Fallback: replace @_all if not already handled by mentions list
	if strings.Contains(text, "@_all") {
		text = strings.ReplaceAll(text, "@_all", "@[所有人]")
	}
	return text
}

func containsBotMention(mentions []*larkim.MentionEvent, botOpenID string) bool {
	if botOpenID == "" {
		return false
	}
	for _, m := range mentions {
		if m.Id != nil && m.Id.OpenId != nil && *m.Id.OpenId == botOpenID {
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
