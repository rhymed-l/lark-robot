package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"lark-robot/internal/model"
	"lark-robot/internal/service"
)

type AutoReplyAPI struct {
	replyService *service.ReplyService
}

func NewAutoReplyAPI(rs *service.ReplyService) *AutoReplyAPI {
	return &AutoReplyAPI{replyService: rs}
}

func (api *AutoReplyAPI) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	rules, total, err := api.replyService.List(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rules, "total": total})
}

func (api *AutoReplyAPI) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	rule, err := api.replyService.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rule})
}

type CreateAutoReplyRequest struct {
	Keyword   string `json:"keyword" binding:"required"`
	ReplyText string `json:"reply_text" binding:"required"`
	MatchMode string `json:"match_mode"`
	ChatID    string `json:"chat_id"`
	Enabled   *bool  `json:"enabled"`
}

func (api *AutoReplyAPI) Create(c *gin.Context) {
	var req CreateAutoReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule := &model.AutoReplyRule{
		Keyword:   req.Keyword,
		ReplyText: req.ReplyText,
		MatchMode: req.MatchMode,
		ChatID:    req.ChatID,
		Enabled:   true,
	}
	if rule.MatchMode == "" {
		rule.MatchMode = "contains"
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}

	if err := api.replyService.Create(rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": rule})
}

func (api *AutoReplyAPI) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	rule, err := api.replyService.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	var req CreateAutoReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule.Keyword = req.Keyword
	rule.ReplyText = req.ReplyText
	if req.MatchMode != "" {
		rule.MatchMode = req.MatchMode
	}
	rule.ChatID = req.ChatID
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}

	if err := api.replyService.Update(rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rule})
}

func (api *AutoReplyAPI) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := api.replyService.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (api *AutoReplyAPI) Toggle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := api.replyService.Toggle(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "toggled"})
}
