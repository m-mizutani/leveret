package adapter

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"google.golang.org/genai"
)

type Gemini interface {
	GenerateContent(ctx context.Context, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error)
	CreateChat(ctx context.Context, config *genai.GenerateContentConfig, history []*genai.Content) (*genai.Chat, error)
	Embedding(ctx context.Context, text string) (*genai.EmbedContentResponse, error)
}

type GeminiClient struct {
	client          *genai.Client
	generativeModel string
	embeddingModel  string
}

type GeminiOption func(*GeminiClient)

func WithGenerativeModel(model string) GeminiOption {
	return func(g *GeminiClient) {
		g.generativeModel = model
	}
}

func WithEmbeddingModel(model string) GeminiOption {
	return func(g *GeminiClient) {
		g.embeddingModel = model
	}
}

func NewGemini(ctx context.Context, projectID, location string, opts ...GeminiOption) (*GeminiClient, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  projectID,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create genai client")
	}

	g := &GeminiClient{
		client:          client,
		generativeModel: "gemini-2.5-flash",
		embeddingModel:  "gemini-embedding-001",
	}

	for _, opt := range opts {
		opt(g)
	}

	return g, nil
}

func (g *GeminiClient) GenerateContent(ctx context.Context, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	resp, err := g.client.Models.GenerateContent(ctx, g.generativeModel, contents, config)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate content")
	}
	return resp, nil
}

func (g *GeminiClient) CreateChat(ctx context.Context, config *genai.GenerateContentConfig, history []*genai.Content) (*genai.Chat, error) {
	chat, err := g.client.Chats.Create(ctx, g.generativeModel, config, history)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create new gemini chat")
	}

	return chat, nil
}

func (g *GeminiClient) Embedding(ctx context.Context, text string) (*genai.EmbedContentResponse, error) {
	resp, err := g.client.Models.EmbedContent(ctx, g.embeddingModel, genai.Text(text), &genai.EmbedContentConfig{})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to embed content")
	}

	return resp, nil
}
