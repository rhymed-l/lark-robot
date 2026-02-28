package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"lark-robot/internal/repository"
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

	users, total, err := api.userService.ListUsers(repository.UserQuery{
		Page:     page,
		PageSize: pageSize,
		Keyword:  keyword,
		SortBy:   c.Query("sort_by"),
		SortDir:  c.Query("sort_dir"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users, "total": total})
}

// Sync re-fetches all users from Lark API and updates the database.
// If a JSON body with "open_ids" is provided, only those users are synced (for retrying).
func (api *UserAPI) Sync(c *gin.Context) {
	var body struct {
		OpenIDs []string `json:"open_ids"`
	}
	_ = c.ShouldBindJSON(&body)

	var result *service.SyncResult
	var err error

	if len(body.OpenIDs) > 0 {
		// Specific users: force sync (bypass 1-hour cooldown)
		result, err = api.userService.SyncByIDs(c.Request.Context(), body.OpenIDs, true)
	} else {
		result, err = api.userService.SyncAllUsers(c.Request.Context())
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetByOpenID returns a user by open_id.
// If the user is not in the database, it fetches from Lark API and persists.
func (api *UserAPI) GetByOpenID(c *gin.Context) {
	openID := c.Param("open_id")
	user, err := api.userService.GetUser(openID)
	if err != nil {
		// Not in DB yet — try fetching from Lark API
		user, err = api.userService.SyncUser(c.Request.Context(), openID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}
