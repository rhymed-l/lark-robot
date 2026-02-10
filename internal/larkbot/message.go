package larkbot

import (
	"context"
	"fmt"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// SendMessage sends a message to a chat or user.
func (c *LarkClient) SendMessage(ctx context.Context, receiveID, receiveIDType, msgType, content string) (string, error) {
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(receiveID).
			MsgType(msgType).
			Content(content).
			Build()).
		Build()

	resp, err := c.Client.Im.Message.Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("send message failed: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("send message error: code=%d, msg=%s", resp.Code, resp.Msg)
	}
	return *resp.Data.MessageId, nil
}

// ReplyMessage replies to a specific message.
func (c *LarkClient) ReplyMessage(ctx context.Context, messageID, msgType, content string) (string, error) {
	req := larkim.NewReplyMessageReqBuilder().
		MessageId(messageID).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(msgType).
			Content(content).
			Build()).
		Build()

	resp, err := c.Client.Im.Message.Reply(ctx, req)
	if err != nil {
		return "", fmt.Errorf("reply message failed: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("reply message error: code=%d, msg=%s", resp.Code, resp.Msg)
	}
	return *resp.Data.MessageId, nil
}

// SendTextMessage is a convenience method for sending plain text.
func (c *LarkClient) SendTextMessage(ctx context.Context, receiveID, receiveIDType, text string) (string, error) {
	content := fmt.Sprintf(`{"text":"%s"}`, text)
	return c.SendMessage(ctx, receiveID, receiveIDType, "text", content)
}
