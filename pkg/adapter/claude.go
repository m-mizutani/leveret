package adapter

import "context"

// Message represents a message in a conversation with Claude
type Message struct {
	Role    string
	Content string
}

// Response represents a response from Claude API
type Response struct {
	Content   string
	ToolCalls []ToolCall
}

// ToolCall represents a tool invocation request from Claude
type ToolCall struct {
	ID       string
	Name     string
	Input    map[string]any
}

// Claude is the interface for Claude API client
type Claude interface {
	// Chat sends messages to Claude and returns response
	Chat(ctx context.Context, messages []Message) (*Response, error)
}

// claudeClient implements Claude interface
type claudeClient struct {
	apiKey string
}

// NewClaude creates a new Claude API client
func NewClaude(apiKey string) Claude {
	return &claudeClient{
		apiKey: apiKey,
	}
}

func (c *claudeClient) Chat(ctx context.Context, messages []Message) (*Response, error) {
	// TODO: Implement actual Claude API integration
	return &Response{}, nil
}
