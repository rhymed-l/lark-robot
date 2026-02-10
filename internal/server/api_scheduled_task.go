package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"lark-robot/internal/model"
	"lark-robot/internal/service"
)

type ScheduledTaskAPI struct {
	schedulerService *service.SchedulerService
}

func NewScheduledTaskAPI(ss *service.SchedulerService) *ScheduledTaskAPI {
	return &ScheduledTaskAPI{schedulerService: ss}
}

func (api *ScheduledTaskAPI) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	tasks, total, err := api.schedulerService.List(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tasks, "total": total})
}

func (api *ScheduledTaskAPI) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	task, err := api.schedulerService.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": task})
}

type CreateScheduledTaskRequest struct {
	Name     string `json:"name" binding:"required"`
	CronExpr string `json:"cron_expr" binding:"required"`
	ChatID   string `json:"chat_id" binding:"required"`
	MsgType  string `json:"msg_type"`
	Content  string `json:"content" binding:"required"`
	Enabled  *bool  `json:"enabled"`
}

func (api *ScheduledTaskAPI) Create(c *gin.Context) {
	var req CreateScheduledTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task := &model.ScheduledTask{
		Name:     req.Name,
		CronExpr: req.CronExpr,
		ChatID:   req.ChatID,
		MsgType:  req.MsgType,
		Content:  req.Content,
		Enabled:  true,
	}
	if task.MsgType == "" {
		task.MsgType = "text"
	}
	if req.Enabled != nil {
		task.Enabled = *req.Enabled
	}

	if err := api.schedulerService.Create(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": task})
}

func (api *ScheduledTaskAPI) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	task, err := api.schedulerService.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	var req CreateScheduledTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task.Name = req.Name
	task.CronExpr = req.CronExpr
	task.ChatID = req.ChatID
	task.Content = req.Content
	if req.MsgType != "" {
		task.MsgType = req.MsgType
	}
	if req.Enabled != nil {
		task.Enabled = *req.Enabled
	}

	if err := api.schedulerService.Update(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": task})
}

func (api *ScheduledTaskAPI) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := api.schedulerService.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (api *ScheduledTaskAPI) Toggle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := api.schedulerService.Toggle(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "toggled"})
}

func (api *ScheduledTaskAPI) RunNow(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := api.schedulerService.RunNow(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "task executed"})
}
