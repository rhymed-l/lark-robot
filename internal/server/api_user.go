package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"lark-robot/internal/service"
)

type UserAPI struct {
	userService *service.UserService
}

func NewUserAPI(us *service.UserService) *UserAPI {
	return &UserAPI{userService: us}
}

// List returns a paginated list of users.
func (api *UserAPI) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	keyword := c.Query("keyword")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	users, total, err := api.userService.ListUsers(page, pageSize, keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users, "total": total})
}

// Sync re-fetches all users from Lark API and updates the database.
func (api *UserAPI) Sync(c *gin.Context) {
	synced, err := api.userService.SyncAllUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "sync completed", "synced": synced})
}

// GetByOpenID returns a user by open_id.
func (api *UserAPI) GetByOpenID(c *gin.Context) {
	openID := c.Param("open_id")
	user, err := api.userService.GetUser(openID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}
