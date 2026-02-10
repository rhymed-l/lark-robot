package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"lark-robot/internal/broadcast"
	"lark-robot/internal/repository"
	"lark-robot/internal/service"
)

type MessageAPI struct {
	messageService *service.MessageService
	broadcaster    *broadcast.MessageBroadcaster
}

func NewMessageAPI(ms *service.MessageService, b *broadcast.MessageBroadcaster) *MessageAPI {
	return &MessageAPI{messageService: ms, broadcaster: b}
}

type SendMessageRequest struct {
	ReceiveID     string `json:"receive_id" binding:"required"`
	ReceiveIDType string `json:"receive_id_type" binding:"required"`
	MsgType       string `json:"msg_type" binding:"required"`
	Content       string `json:"content" binding:"required"`
}

func (api *MessageAPI) Send(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msgID, err := api.messageService.SendMessage(c.Request.Context(), req.ReceiveID, req.ReceiveIDType, req.MsgType, req.Content, "manual")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message_id": msgID})
}

func (api *MessageAPI) GetLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	q := repository.MessageLogQuery{
		ChatID:    c.Query("chat_id"),
		ChatType:  c.Query("chat_type"),
		Direction: c.Query("direction"),
		Source:    c.Query("source"),
		Page:     page,
		PageSize: pageSize,
	}

	logs, total, err := api.messageService.GetLogs(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  logs,
		"total": total,
		"page":  page,
	})
}

// Conversations returns distinct chat_ids from message logs for the chat sidebar.
func (api *MessageAPI) Conversations(c *gin.Context) {
	conversations, err := api.messageService.ListConversations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": conversations})
}

// Stream provides a Server-Sent Events endpoint for real-time message updates.
// If chat_id is empty, subscribes to all messages (global notifications).
func (api *MessageAPI) Stream(c *gin.Context) {
	chatID := c.Query("chat_id") // empty = global subscription

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Subscribe (empty chatID = global)
	ch := api.broadcaster.Subscribe(chatID)
	defer api.broadcaster.Unsubscribe(chatID, ch)

	// Get the client gone channel
	clientGone := c.Request.Context().Done()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-clientGone:
			return false
		case event, ok := <-ch:
			if !ok {
				return false
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			return true
		case <-ticker.C:
			// Heartbeat to keep connection alive
			fmt.Fprintf(w, ": ping\n\n")
			return true
		}
	})
}
