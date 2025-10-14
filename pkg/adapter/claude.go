package adapter

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Claude is the interface for Claude API client
type Claude interface {
	// Chat sends messages to Claude and returns response
	Chat(ctx context.Context, messages []anthropic.MessageParam) (*anthropic.Message, error)
}

// claudeClient implements Claude interface
type claudeClient struct {
	client *anthropic.Client
}

// NewClaude creates a new Claude API client
func NewClaude(apiKey string) Claude {
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)
	return &claudeClient{
		client: &client,
	}
}

func (c *claudeClient) Chat(ctx context.Context, messages []anthropic.MessageParam) (*anthropic.Message, error) {
	// TODO: Implement actual Claude API integration
	// message, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
	//     Model:      anthropic.F(anthropic.ModelClaude_3_5_Sonnet_20241022),
	//     Messages:   anthropic.F(messages),
	//     MaxTokens:  anthropic.F(int64(1024)),
	// })
	return nil, nil
}
