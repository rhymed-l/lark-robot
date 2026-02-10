package handler

import "context"

// DefaultHandler is a fallback handler at the end of the chain.
// It does not produce a reply — it simply marks the message as handled
// so the chain terminates cleanly.
type DefaultHandler struct{}

func NewDefaultHandler() *DefaultHandler {
	return &DefaultHandler{}
}

func (h *DefaultHandler) Name() string { return "DefaultHandler" }

func (h *DefaultHandler) Handle(ctx context.Context, msg *IncomingMessage) (*Result, error) {
	// Silently consume unhandled messages — no reply.
	return &Result{Handled: true, Reply: nil}, nil
}
