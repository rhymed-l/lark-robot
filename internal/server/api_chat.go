package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"lark-robot/internal/service"
)

type ChatAPI struct {
	chatService *service.ChatService
}

func NewChatAPI(cs *service.ChatService) *ChatAPI {
	return &ChatAPI{chatService: cs}
}

func (api *ChatAPI) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	groups, total, err := api.chatService.ListGroups(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": groups, "total": total})
}

func (api *ChatAPI) Sync(c *gin.Context) {
	groups, err := api.chatService.SyncChats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": groups, "message": "sync completed"})
}

func (api *ChatAPI) Leave(c *gin.Context) {
	chatID := c.Param("chat_id")
	if err := api.chatService.LeaveChat(c.Request.Context(), chatID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "left chat successfully"})
}
