package larkbot

import (
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
)

type LarkClient struct {
	Client *lark.Client
	AppID  string
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
