package server

import (
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"lark-robot/internal/broadcast"
	"lark-robot/internal/service"
)

type Router struct {
	Engine           *gin.Engine
	logger           *zap.Logger
	authAPI          *AuthAPI
	dashboardAPI     *DashboardAPI
	messageAPI       *MessageAPI
	chatAPI          *ChatAPI
	autoReplyAPI     *AutoReplyAPI
	scheduledTaskAPI *ScheduledTaskAPI
	authSecret       string
	frontendFS       http.FileSystem
}

type RouterConfig struct {
	Mode             string
	Logger           *zap.Logger
	AuthUsername      string
	AuthPassword      string
	AuthSecret        string
	ChatService      *service.ChatService
	MessageService   *service.MessageService
	SchedulerService *service.SchedulerService
	ReplyService     *service.ReplyService
	Broadcaster      *broadcast.MessageBroadcaster
	FrontendFS       http.FileSystem
}

func NewRouter(cfg RouterConfig) *Router {
	if cfg.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := &Router{
		Engine:             gin.New(),
		logger:             cfg.Logger,
		authAPI:            NewAuthAPI(cfg.AuthUsername, cfg.AuthPassword, cfg.AuthSecret),
		dashboardAPI:       NewDashboardAPI(cfg.ChatService, cfg.MessageService, cfg.SchedulerService, cfg.ReplyService),
		messageAPI:         NewMessageAPI(cfg.MessageService, cfg.Broadcaster),
		chatAPI:            NewChatAPI(cfg.ChatService),
		autoReplyAPI:       NewAutoReplyAPI(cfg.ReplyService),
		scheduledTaskAPI:   NewScheduledTaskAPI(cfg.SchedulerService),
		authSecret:         cfg.AuthSecret,
		frontendFS:         cfg.FrontendFS,
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
		// Dashboard
		authed.GET("/dashboard/stats", r.dashboardAPI.GetStats)

		// Messages
		authed.POST("/messages/send", r.messageAPI.Send)
		authed.GET("/messages/logs", r.messageAPI.GetLogs)
		authed.GET("/messages/conversations", r.messageAPI.Conversations)
		authed.GET("/messages/stream", r.messageAPI.Stream)

		// Chats (groups)
		authed.GET("/chats", r.chatAPI.List)
		authed.POST("/chats/sync", r.chatAPI.Sync)
		authed.POST("/chats/:chat_id/leave", r.chatAPI.Leave)

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
		r.Engine.StaticFS("/assets", r.frontendFS)
		r.Engine.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			// SPA fallback: serve index.html for non-API routes
			file, err := r.frontendFS.Open("index.html")
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "frontend not found"})
				return
			}
			defer file.Close()
			stat, _ := file.Stat()
			content, _ := io.ReadAll(file)
			http.ServeContent(c.Writer, c.Request, "index.html", stat.ModTime(), strings.NewReader(string(content)))
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
func TryLoadFrontendFS(fsys fs.FS) http.FileSystem {
	if fsys == nil {
		return nil
	}
	sub, err := fs.Sub(fsys, "dist")
	if err != nil {
		return nil
	}
	// Check if dist directory has content
	entries, err := fs.ReadDir(sub, ".")
	if err != nil || len(entries) == 0 {
		return nil
	}
	return http.FS(sub)
}
