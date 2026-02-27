package broadcast

import (
	"sync"
	"time"
)

// MessageEvent is sent to SSE subscribers when a message is received or sent.
type MessageEvent struct {
	ID        string    `json:"id"`
	ChatID    string    `json:"chat_id"`
	ChatType  string    `json:"chat_type"` // "p2p" or "group"
	SenderID   string    `json:"sender_id"`
	SenderName string    `json:"sender_name"`
	Direction string    `json:"direction"` // "in" or "out"
	MsgType   string    `json:"msg_type"`
	Content   string    `json:"content"`
	Recalled  bool      `json:"recalled,omitempty"`
	MessageID string    `json:"message_id,omitempty"` // target message_id for recall events
	CreatedAt time.Time `json:"created_at"`
}

// MessageBroadcaster manages SSE subscriber channels per chat.
type MessageBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan MessageEvent]struct{}
}

func NewMessageBroadcaster() *MessageBroadcaster {
	return &MessageBroadcaster{
		subscribers: make(map[string]map[chan MessageEvent]struct{}),
	}
}

// Subscribe registers a new listener for a specific chat and returns a channel.
func (b *MessageBroadcaster) Subscribe(chatID string) chan MessageEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan MessageEvent, 32)
	if b.subscribers[chatID] == nil {
		b.subscribers[chatID] = make(map[chan MessageEvent]struct{})
	}
	b.subscribers[chatID][ch] = struct{}{}
	return ch
}

// Unsubscribe removes a listener for a specific chat.
func (b *MessageBroadcaster) Unsubscribe(chatID string, ch chan MessageEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if subs, ok := b.subscribers[chatID]; ok {
		delete(subs, ch)
		close(ch)
		if len(subs) == 0 {
			delete(b.subscribers, chatID)
		}
	}
}

// Publish sends a message event to chat-specific subscribers and global subscribers (key="").
func (b *MessageBroadcaster) Publish(event MessageEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Send to chat-specific subscribers
	if subs, ok := b.subscribers[event.ChatID]; ok {
		for ch := range subs {
			select {
			case ch <- event:
			default:
			}
		}
	}

	// Send to global subscribers (subscribed with empty chatID)
	if subs, ok := b.subscribers[""]; ok {
		for ch := range subs {
			select {
			case ch <- event:
			default:
			}
		}
	}
}
