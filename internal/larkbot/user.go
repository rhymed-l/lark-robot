package larkbot

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
)

// UserInfo holds user information from the Lark API.
type UserInfo struct {
	OpenID          string
	UnionID         string
	UserID          string
	Name            string
	EnName          string
	Avatar          string
	Description     string
	Email           string
	City            string
	JobTitle        string
	WorkStation     string
	EmployeeNo      string
	Gender          int
	LeaderUserID    string
	DepartmentIDs   string // JSON array string of department IDs
	DepartmentNames string // JSON array string of department names
	CustomAttrs     string // JSON string
	JoinTime        int64
}

// department name cache
var (
	deptCache   = make(map[string]string)
	deptCacheMu sync.RWMutex
)

// GetDepartmentName returns the department name for a given department ID.
func (c *LarkClient) GetDepartmentName(ctx context.Context, deptID string) (string, error) {
	if deptID == "" || deptID == "0" {
		return "", nil
	}

	deptCacheMu.RLock()
	if name, ok := deptCache[deptID]; ok {
		deptCacheMu.RUnlock()
		return name, nil
	}
	deptCacheMu.RUnlock()

	req := larkcontact.NewGetDepartmentReqBuilder().
		DepartmentId(deptID).
		DepartmentIdType("open_department_id").
		Build()

	resp, err := c.Client.Contact.Department.Get(ctx, req)
	if err != nil {
		// Cache the fallback to avoid repeated failed API calls
		deptCacheMu.Lock()
		deptCache[deptID] = deptID
		deptCacheMu.Unlock()
		return deptID, fmt.Errorf("get department failed: %w", err)
	}
	if !resp.Success() {
		deptCacheMu.Lock()
		deptCache[deptID] = deptID
		deptCacheMu.Unlock()
		return deptID, fmt.Errorf("get department error: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	name := deref(resp.Data.Department.Name)
	if name == "" {
		name = deptID
	}

	deptCacheMu.Lock()
	deptCache[deptID] = name
	deptCacheMu.Unlock()

	return name, nil
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
	if user.Description != nil {
		info.Description = *user.Description
	}
	if user.Email != nil {
		info.Email = *user.Email
	}
	if user.City != nil {
		info.City = *user.City
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
	if len(user.DepartmentIds) > 0 {
		if b, err := json.Marshal(user.DepartmentIds); err == nil {
			info.DepartmentIDs = string(b)
		}
		// Resolve department names
		var names []string
		for _, deptID := range user.DepartmentIds {
			name, err := c.GetDepartmentName(ctx, deptID)
			if err != nil {
				names = append(names, deptID)
			} else {
				names = append(names, name)
			}
		}
		if b, err := json.Marshal(names); err == nil {
			info.DepartmentNames = string(b)
		}
	}
	if len(user.CustomAttrs) > 0 {
		if b, err := json.Marshal(user.CustomAttrs); err == nil {
			info.CustomAttrs = string(b)
		}
	}
	if user.JoinTime != nil {
		info.JoinTime = int64(*user.JoinTime)
	}

	return info, nil
}
