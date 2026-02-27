package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"lark-robot/internal/service"
)

type DashboardAPI struct {
	chatService      *service.ChatService
	messageService   *service.MessageService
	schedulerService *service.SchedulerService
	replyService     *service.ReplyService
	userService      *service.UserService
}

func NewDashboardAPI(cs *service.ChatService, ms *service.MessageService, ss *service.SchedulerService, rs *service.ReplyService, us *service.UserService) *DashboardAPI {
	return &DashboardAPI{
		chatService:      cs,
		messageService:   ms,
		schedulerService: ss,
		replyService:     rs,
		userService:      us,
	}
}

func (api *DashboardAPI) GetStats(c *gin.Context) {
	groupCount, _ := api.chatService.GroupCount()
	messagesToday, _ := api.messageService.CountToday()
	taskCount, _ := api.schedulerService.TaskCount()
	userCount, _ := api.userService.UserCount()

	_, ruleCount, _ := api.replyService.List(1, 1)

	c.JSON(http.StatusOK, gin.H{
		"group_count":    groupCount,
		"messages_today": messagesToday,
		"task_count":     taskCount,
		"rule_count":     ruleCount,
		"user_count":     userCount,
	})
}
