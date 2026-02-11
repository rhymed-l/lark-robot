package larkbot

import (
	"context"
	"encoding/json"
	"fmt"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
)

type LarkClient struct {
	Client    *lark.Client
	AppID     string
	BotOpenID string // Bot's own open_id, fetched at startup
}

func NewLarkClient(appID, appSecret, baseURL string) *LarkClient {
	opts := []lark.ClientOptionFunc{
		lark.WithEnableTokenCache(true),
		lark.WithLogLevel(larkcore.LogLevelInfo),
	}
	if baseURL != "" {
		opts = append(opts, lark.WithOpenBaseUrl(baseURL))
	}
	client := lark.NewClient(appID, appSecret, opts...)
	return &LarkClient{Client: client, AppID: appID}
}

// FetchBotInfo retrieves the bot's own open_id via the Lark REST API.
func (c *LarkClient) FetchBotInfo(ctx context.Context) error {
	resp, err := c.Client.Get(ctx, "/open-apis/bot/v3/info", nil, larkcore.AccessTokenTypeTenant)
	if err != nil {
		return fmt.Errorf("get bot info failed: %w", err)
	}

	var result struct {
		Bot struct {
			OpenID string `json:"open_id"`
		} `json:"bot"`
	}
	if err := json.Unmarshal(resp.RawBody, &result); err != nil {
		return fmt.Errorf("parse bot info failed: %w", err)
	}
	if result.Bot.OpenID != "" {
		c.BotOpenID = result.Bot.OpenID
	}
	return nil
}
