package adapter

import "context"

// Gemini is the interface for Gemini API client
type Gemini interface {
	// GenerateEmbedding generates embedding vector from text
	GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
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
	// TODO: Implement actual Gemini API integration
	return []float64{}, nil
}
