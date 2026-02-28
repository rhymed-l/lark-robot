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

type ReplyMessageRequest struct {
	MessageID string `json:"message_id" binding:"required"`
	MsgType   string `json:"msg_type" binding:"required"`
	Content   string `json:"content" binding:"required"`
}

func (api *MessageAPI) Reply(c *gin.Context) {
	var req ReplyMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msgID, err := api.messageService.ReplyMessage(c.Request.Context(), req.MessageID, req.MsgType, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message_id": msgID})
}

func (api *MessageAPI) Delete(c *gin.Context) {
	messageID := c.Param("message_id")
	if messageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message_id is required"})
		return
	}

	if err := api.messageService.DeleteMessage(c.Request.Context(), messageID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (api *MessageAPI) GetImage(c *gin.Context) {
	messageID := c.Param("message_id")
	fileKey := c.Param("file_key")
	if messageID == "" || fileKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message_id and file_key are required"})
		return
	}

	resType := c.DefaultQuery("type", "image")
	reader, err := api.messageService.GetMessageResource(c.Request.Context(), messageID, fileKey, resType)
	if err != nil {
		fmt.Printf("[GetImage] error for msg=%s key=%s: %v\n", messageID, fileKey, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Read first 512 bytes for content type detection
	buf := make([]byte, 512)
	n, readErr := reader.Read(buf)
	if n == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "empty image response"})
		return
	}

	contentType := http.DetectContentType(buf[:n])
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=86400")

	// For file downloads, set Content-Disposition to trigger browser download
	if resType == "file" {
		fileName := c.Query("filename")
		if fileName == "" {
			fileName = fileKey
		}
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	}

	c.Writer.Write(buf[:n])
	if readErr != io.EOF {
		io.Copy(c.Writer, reader)
	}
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
