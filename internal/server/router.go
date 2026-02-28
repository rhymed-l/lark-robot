package server

import (
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"lark-robot/internal/broadcast"
	"lark-robot/internal/larkbot"
	"lark-robot/internal/service"
)

type Router struct {
	Engine           *gin.Engine
	logger           *zap.Logger
	authAPI          *AuthAPI
	dashboardAPI     *DashboardAPI
	messageAPI       *MessageAPI
	uploadAPI        *UploadAPI
	chatAPI          *ChatAPI
	userAPI          *UserAPI
	autoReplyAPI     *AutoReplyAPI
	scheduledTaskAPI *ScheduledTaskAPI
	larkClient       *larkbot.LarkClient
	authSecret       string
	frontendFS       http.FileSystem
	embeddedFS       fs.FS
}

type RouterConfig struct {
	Mode             string
	Logger           *zap.Logger
	AuthUsername      string
	AuthPassword      string
	AuthSecret        string
	LarkClient       *larkbot.LarkClient
	ChatService      *service.ChatService
	MessageService   *service.MessageService
	SchedulerService *service.SchedulerService
	ReplyService     *service.ReplyService
	UserService      *service.UserService
	Broadcaster      *broadcast.MessageBroadcaster
	FrontendFS       http.FileSystem
	EmbeddedFS       fs.FS
}

func NewRouter(cfg RouterConfig) *Router {
	if cfg.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := &Router{
		Engine:             gin.New(),
		logger:             cfg.Logger,
		authAPI:            NewAuthAPI(cfg.AuthUsername, cfg.AuthPassword, cfg.AuthSecret),
		dashboardAPI:       NewDashboardAPI(cfg.ChatService, cfg.MessageService, cfg.SchedulerService, cfg.ReplyService, cfg.UserService),
		messageAPI:         NewMessageAPI(cfg.MessageService, cfg.Broadcaster),
		uploadAPI:          NewUploadAPI(cfg.LarkClient),
		chatAPI:            NewChatAPI(cfg.ChatService),
		userAPI:            NewUserAPI(cfg.UserService),
		autoReplyAPI:       NewAutoReplyAPI(cfg.ReplyService),
		scheduledTaskAPI:   NewScheduledTaskAPI(cfg.SchedulerService),
		larkClient:         cfg.LarkClient,
		authSecret:         cfg.AuthSecret,
		frontendFS:         cfg.FrontendFS,
		embeddedFS:         cfg.EmbeddedFS,
	}

	r.setupRoutes()
	return r
}

func (r *Router) setupRoutes() {
	r.Engine.Use(gin.Recovery())
	r.Engine.Use(CORSMiddleware())
	r.Engine.Use(LoggerMiddleware(r.logger))

	api := r.Engine.Group("/api")
	{
		// Login (no auth required)
		api.POST("/login", r.authAPI.Login)
	}

	// All other API routes require authentication
	authed := r.Engine.Group("/api")
	authed.Use(AuthMiddleware(r.authSecret))
	{
		// Bot info
		authed.GET("/bot/info", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"name":       r.larkClient.BotName,
				"open_id":    r.larkClient.BotOpenID,
				"avatar_url": r.larkClient.BotAvatarURL,
			})
		})

		// Dashboard
		authed.GET("/dashboard/stats", r.dashboardAPI.GetStats)

		// Messages
		authed.POST("/messages/send", r.messageAPI.Send)
		authed.POST("/messages/reply", r.messageAPI.Reply)
		authed.DELETE("/messages/:message_id", r.messageAPI.Delete)
		authed.GET("/messages/logs", r.messageAPI.GetLogs)
		authed.GET("/messages/conversations", r.messageAPI.Conversations)
		authed.GET("/messages/stream", r.messageAPI.Stream)
		authed.GET("/images/:message_id/:file_key", r.messageAPI.GetImage)

		// Upload
		authed.POST("/upload/image", r.uploadAPI.UploadImage)
		authed.POST("/upload/file", r.uploadAPI.UploadFile)

		// Chats (groups)
		authed.GET("/chats", r.chatAPI.List)
		authed.POST("/chats/sync", r.chatAPI.Sync)
		authed.POST("/chats/:chat_id/leave", r.chatAPI.Leave)
		authed.GET("/chats/:chat_id/members", r.chatAPI.Members)

		// Users
		authed.GET("/users", r.userAPI.List)
		authed.POST("/users/sync", r.userAPI.Sync)
		authed.GET("/users/:open_id", r.userAPI.GetByOpenID)

		// Auto-reply rules
		rules := authed.Group("/auto-reply-rules")
		{
			rules.GET("", r.autoReplyAPI.List)
			rules.POST("", r.autoReplyAPI.Create)
			rules.GET("/:id", r.autoReplyAPI.GetByID)
			rules.PUT("/:id", r.autoReplyAPI.Update)
			rules.DELETE("/:id", r.autoReplyAPI.Delete)
			rules.POST("/:id/toggle", r.autoReplyAPI.Toggle)
		}

		// Scheduled tasks
		tasks := authed.Group("/scheduled-tasks")
		{
			tasks.GET("", r.scheduledTaskAPI.List)
			tasks.POST("", r.scheduledTaskAPI.Create)
			tasks.GET("/:id", r.scheduledTaskAPI.GetByID)
			tasks.PUT("/:id", r.scheduledTaskAPI.Update)
			tasks.DELETE("/:id", r.scheduledTaskAPI.Delete)
			tasks.POST("/:id/toggle", r.scheduledTaskAPI.Toggle)
			tasks.POST("/:id/run", r.scheduledTaskAPI.RunNow)
		}

	}

	// Serve frontend
	if r.frontendFS != nil {
		// Read index.html into memory for SPA fallback
		indexFile, _ := r.frontendFS.Open("index.html")
		indexBytes := make([]byte, 0)
		if indexFile != nil {
			indexBytes, _ = io.ReadAll(indexFile)
			indexFile.Close()
		}

		// Serve static assets (js, css, images) under /assets/
		assetsFS, _ := fs.Sub(r.embeddedFS, "assets")
		r.Engine.StaticFS("/assets", http.FS(assetsFS))

		// SPA fallback for all other non-API routes
		r.Engine.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", indexBytes)
		})
	} else {
		r.Engine.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "frontend not embedded, run in dev mode"})
		})
	}
}

// TryLoadFrontendFS attempts to load the embedded frontend filesystem.
// fsys should already point to the dist directory (via static.DistFS()).
func TryLoadFrontendFS(fsys fs.FS) http.FileSystem {
	if fsys == nil {
		return nil
	}
	// Check if directory has content (more than just .gitkeep)
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil || len(entries) <= 1 {
		return nil
	}
	return http.FS(fsys)
}
