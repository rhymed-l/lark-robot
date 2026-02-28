package larkbot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
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

// GetMessageResource downloads a resource (image/file) from a message.
// resType should be "image" or "file".
func (c *LarkClient) GetMessageResource(ctx context.Context, messageID, fileKey, resType string) (io.Reader, error) {
	if resType == "" {
		resType = "image"
	}
	apiPath := fmt.Sprintf("/open-apis/im/v1/messages/%s/resources/%s?type=%s", messageID, fileKey, resType)
	resp, err := c.Client.Get(ctx, apiPath, nil, larkcore.AccessTokenTypeTenant)
	if err != nil {
		return nil, fmt.Errorf("get message resource failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get message resource error: status=%d, body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if len(resp.RawBody) == 0 {
		return nil, fmt.Errorf("get message resource error: empty response")
	}
	return bytes.NewReader(resp.RawBody), nil
}

// DeleteMessage deletes (recalls) a message by its message ID.
func (c *LarkClient) DeleteMessage(ctx context.Context, messageID string) error {
	req := larkim.NewDeleteMessageReqBuilder().
		MessageId(messageID).
		Build()

	resp, err := c.Client.Im.Message.Delete(ctx, req)
	if err != nil {
		return fmt.Errorf("delete message failed: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("delete message error: code=%d, msg=%s", resp.Code, resp.Msg)
	}
	return nil
}

// UploadImage uploads an image to Lark and returns the image_key.
func (c *LarkClient) UploadImage(ctx context.Context, imageReader io.Reader) (string, error) {
	req := larkim.NewCreateImageReqBuilder().
		Body(larkim.NewCreateImageReqBodyBuilder().
			ImageType("message").
			Image(imageReader).
			Build()).
		Build()

	resp, err := c.Client.Im.Image.Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("upload image failed: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("upload image error: code=%d, msg=%s", resp.Code, resp.Msg)
	}
	return *resp.Data.ImageKey, nil
}

// UploadFile uploads a file to Lark and returns the file_key.
// fileType: opus/mp4/pdf/doc/xls/ppt/stream
func (c *LarkClient) UploadFile(ctx context.Context, fileType, fileName string, fileReader io.Reader) (string, error) {
	req := larkim.NewCreateFileReqBuilder().
		Body(larkim.NewCreateFileReqBodyBuilder().
			FileType(fileType).
			FileName(fileName).
			File(fileReader).
			Build()).
		Build()

	resp, err := c.Client.Im.File.Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("upload file failed: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("upload file error: code=%d, msg=%s", resp.Code, resp.Msg)
	}
	return *resp.Data.FileKey, nil
}

// SendTextMessage is a convenience method for sending plain text.
func (c *LarkClient) SendTextMessage(ctx context.Context, receiveID, receiveIDType, text string) (string, error) {
	content := fmt.Sprintf(`{"text":"%s"}`, text)
	return c.SendMessage(ctx, receiveID, receiveIDType, "text", content)
}
