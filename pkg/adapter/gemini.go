package adapter

import (
	"context"

	"google.golang.org/genai"
)

// Gemini is the interface for Gemini API client
type Gemini interface {
	// GenerateEmbedding generates embedding vector from text
	GenerateEmbedding(ctx context.Context, text string) ([]float64, error)

	// Chat sends messages to Gemini and returns response
	Chat(ctx context.Context, contents []*genai.Content) (*genai.GenerateContentResponse, error)
}

// geminiClient implements Gemini interface
type geminiClient struct {
	projectID string
	location  string
}

// NewGemini creates a new Gemini API client
func NewGemini(projectID, location string) Gemini {
	return &geminiClient{
		projectID: projectID,
		location:  location,
	}
}

func (g *geminiClient) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	// TODO: Implement actual Gemini API integration for embeddings
	return []float64{}, nil
}

func (g *geminiClient) Chat(ctx context.Context, contents []*genai.Content) (*genai.GenerateContentResponse, error) {
	// TODO: Implement actual Gemini API integration for chat
	return nil, nil
}
