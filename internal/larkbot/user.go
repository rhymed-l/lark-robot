package larkbot

import (
	"context"
	"fmt"
	"sync"

	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
)

// UserInfo holds basic user information.
type UserInfo struct {
	OpenID string
	Name   string
	Avatar string
}

// userCache caches user info to avoid repeated API calls.
var (
	userCache   = make(map[string]*UserInfo)
	userCacheMu sync.RWMutex
)

// GetUserInfo retrieves user info by open_id, with in-memory caching.
func (c *LarkClient) GetUserInfo(ctx context.Context, openID string) (*UserInfo, error) {
	if openID == "" {
		return &UserInfo{OpenID: "", Name: "未知"}, nil
	}

	// Check cache
	userCacheMu.RLock()
	if info, ok := userCache[openID]; ok {
		userCacheMu.RUnlock()
		return info, nil
	}
	userCacheMu.RUnlock()

	// Query Lark API
	req := larkcontact.NewGetUserReqBuilder().
		UserId(openID).
		UserIdType("open_id").
		Build()

	resp, err := c.Client.Contact.User.Get(ctx, req)
	if err != nil {
		return &UserInfo{OpenID: openID, Name: openID}, fmt.Errorf("get user info failed: %w", err)
	}
	if !resp.Success() {
		return &UserInfo{OpenID: openID, Name: openID}, fmt.Errorf("get user info error: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	info := &UserInfo{
		OpenID: openID,
		Name:   deref(resp.Data.User.Name),
	}
	if resp.Data.User.Avatar != nil {
		info.Avatar = deref(resp.Data.User.Avatar.AvatarOrigin)
	}

	// Store in cache
	userCacheMu.Lock()
	userCache[openID] = info
	userCacheMu.Unlock()

	return info, nil
}

// GetCachedUserName returns cached user name, or the openID itself if not cached.
func GetCachedUserName(openID string) string {
	userCacheMu.RLock()
	defer userCacheMu.RUnlock()
	if info, ok := userCache[openID]; ok {
		return info.Name
	}
	return openID
}
