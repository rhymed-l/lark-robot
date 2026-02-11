package handler

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
)

// KeywordRule defines a single keyword-to-reply mapping.
type KeywordRule struct {
	ID          uint
	Keyword     string
	ReplyText   string
	MatchMode   string // "exact", "contains", "prefix"
	ChatID      string // empty = all chats
	TriggerMode string // "any", "at_bot", "p2p_only"
	Enabled     bool
}

// KeywordHandler checks incoming text against a set of keyword rules.
type KeywordHandler struct {
	mu    sync.RWMutex
	rules []KeywordRule
}

func NewKeywordHandler(rules []KeywordRule) *KeywordHandler {
	return &KeywordHandler{rules: rules}
}

func (h *KeywordHandler) Name() string { return "KeywordHandler" }

func (h *KeywordHandler) UpdateRules(rules []KeywordRule) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.rules = rules
}

func (h *KeywordHandler) Handle(ctx context.Context, msg *IncomingMessage) (*Result, error) {
	// Support both plain text and rich text (post) messages
	if msg.MsgType != "text" && msg.MsgType != "post" {
		return &Result{Handled: false}, nil
	}
	if msg.TextContent == "" {
		return &Result{Handled: false}, nil
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, rule := range h.rules {
		if !rule.Enabled {
			continue
		}
		if rule.ChatID != "" && !matchChatID(rule.ChatID, msg.ChatID) {
			continue
		}
		// Check trigger mode
		switch rule.TriggerMode {
		case "at_bot":
			if msg.ChatType != "p2p" && !msg.MentionBot {
				continue
			}
		case "p2p_only":
			if msg.ChatType != "p2p" {
				continue
			}
		}
		if matchKeyword(msg.TextContent, rule.Keyword, rule.MatchMode) {
			replyText := renderTemplate(rule.ReplyText, msg)
			content, _ := json.Marshal(map[string]string{"text": replyText})
			return &Result{
				Handled: true,
				Reply: &Reply{
					MsgType: "text",
					Content: string(content),
				},
			}, nil
		}
	}
	return &Result{Handled: false}, nil
}

// renderTemplate replaces template variables in reply text with actual message values.
// Supported variables: {{chat_id}}, {{chat_type}}, {{sender_id}}, {{sender_name}}, {{message_id}}, {{content}}
func renderTemplate(text string, msg *IncomingMessage) string {
	r := strings.NewReplacer(
		"{{chat_id}}", msg.ChatID,
		"{{chat_type}}", msg.ChatType,
		"{{sender_id}}", msg.SenderID,
		"{{sender_name}}", msg.SenderName,
		"{{message_id}}", msg.MessageID,
		"{{content}}", msg.TextContent,
	)
	return r.Replace(text)
}

// matchChatID checks if msgChatID is in the comma-separated ruleChatID list.
func matchChatID(ruleChatID, msgChatID string) bool {
	for _, id := range strings.Split(ruleChatID, ",") {
		if strings.TrimSpace(id) == msgChatID {
			return true
		}
	}
	return false
}

func matchKeyword(text, keyword, mode string) bool {
	text = strings.TrimSpace(text)
	switch mode {
	case "exact":
		return strings.EqualFold(text, keyword)
	case "prefix":
		return strings.HasPrefix(strings.ToLower(text), strings.ToLower(keyword))
	default: // "contains"
		return strings.Contains(strings.ToLower(text), strings.ToLower(keyword))
	}
}
