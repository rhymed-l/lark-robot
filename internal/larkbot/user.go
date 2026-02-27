package larkbot

import (
	"context"
	"fmt"

	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
)

// UserInfo holds user information from the Lark API.
type UserInfo struct {
	OpenID       string
	UnionID      string
	UserID       string
	Name         string
	EnName       string
	Avatar       string
	Email        string
	JobTitle     string
	WorkStation  string
	EmployeeNo   string
	Gender       int
	LeaderUserID string
	JoinTime     int64
}

// GetUserInfo retrieves user info by open_id via the Lark API.
// Caching is managed externally by UserService.
func (c *LarkClient) GetUserInfo(ctx context.Context, openID string) (*UserInfo, error) {
	if openID == "" {
		return &UserInfo{OpenID: "", Name: "未知"}, nil
	}

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

	user := resp.Data.User
	info := &UserInfo{
		OpenID: openID,
		Name:   deref(user.Name),
		EnName: deref(user.EnName),
	}

	if user.UnionId != nil {
		info.UnionID = *user.UnionId
	}
	if user.UserId != nil {
		info.UserID = *user.UserId
	}
	if user.Avatar != nil {
		info.Avatar = deref(user.Avatar.AvatarOrigin)
	}
	if user.Email != nil {
		info.Email = *user.Email
	}
	if user.JobTitle != nil {
		info.JobTitle = *user.JobTitle
	}
	if user.WorkStation != nil {
		info.WorkStation = *user.WorkStation
	}
	if user.EmployeeNo != nil {
		info.EmployeeNo = *user.EmployeeNo
	}
	if user.Gender != nil {
		info.Gender = *user.Gender
	}
	if user.LeaderUserId != nil {
		info.LeaderUserID = *user.LeaderUserId
	}
	if user.JoinTime != nil {
		info.JoinTime = int64(*user.JoinTime)
	}

	return info, nil
}
