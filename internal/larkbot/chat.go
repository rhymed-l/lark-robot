package larkbot

import (
	"context"
	"fmt"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// ChatInfo represents basic information about a chat/group.
type ChatInfo struct {
	ChatID      string
	Name        string
	Description string
	OwnerID     string
	MemberCount int
	External    bool
}

// ListChats returns all chats the bot has joined.
func (c *LarkClient) ListChats(ctx context.Context) ([]ChatInfo, error) {
	var chats []ChatInfo
	var pageToken *string

	for {
		reqBuilder := larkim.NewListChatReqBuilder().PageSize(100)
		if pageToken != nil {
			reqBuilder.PageToken(*pageToken)
		}
		req := reqBuilder.Build()

		resp, err := c.Client.Im.Chat.List(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("list chats failed: %w", err)
		}
		if !resp.Success() {
			return nil, fmt.Errorf("list chats error: code=%d, msg=%s", resp.Code, resp.Msg)
		}

		for _, item := range resp.Data.Items {
			chat := ChatInfo{
				ChatID: deref(item.ChatId),
				Name:   deref(item.Name),
			}
			if item.Description != nil {
				chat.Description = *item.Description
			}
			if item.OwnerIdType != nil {
				chat.OwnerID = deref(item.OwnerId)
			}
			if item.External != nil {
				chat.External = *item.External
			}
			chats = append(chats, chat)
		}

		if !*resp.Data.HasMore {
			break
		}
		pageToken = resp.Data.PageToken
	}

	return chats, nil
}

// LeaveChat removes the bot itself from a specific chat by deleting its own membership.
func (c *LarkClient) LeaveChat(ctx context.Context, chatID string) error {
	req := larkim.NewDeleteChatMembersReqBuilder().
		ChatId(chatID).
		MemberIdType("app_id").
		Body(larkim.NewDeleteChatMembersReqBodyBuilder().
			IdList([]string{c.AppID}).
			Build()).
		Build()

	resp, err := c.Client.Im.V1.ChatMembers.Delete(ctx, req)
	if err != nil {
		return fmt.Errorf("leave chat failed: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("leave chat error: code=%d, msg=%s", resp.Code, resp.Msg)
	}
	return nil
}

// GetChatInfo retrieves detailed information about a specific chat.
func (c *LarkClient) GetChatInfo(ctx context.Context, chatID string) (*ChatInfo, error) {
	req := larkim.NewGetChatReqBuilder().
		ChatId(chatID).
		Build()

	resp, err := c.Client.Im.Chat.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get chat info failed: %w", err)
	}
	if !resp.Success() {
		return nil, fmt.Errorf("get chat info error: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	return &ChatInfo{
		ChatID:      chatID,
		Name:        deref(resp.Data.Name),
		Description: deref(resp.Data.Description),
		OwnerID:     deref(resp.Data.OwnerId),
		External:    resp.Data.External != nil && *resp.Data.External,
	}, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
