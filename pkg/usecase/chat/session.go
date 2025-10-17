package chat

import (
	"context"

	"github.com/m-mizutani/leveret/pkg/adapter"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/repository"
	"google.golang.org/genai"
)

// Session manages an interactive chat session for alert analysis
type Session struct {
	repo    repository.Repository
	gemini  adapter.Gemini
	storage adapter.Storage
	alertID model.AlertID
	alert   *model.Alert
	history *model.History
}

// NewInput contains parameters for creating a new chat session
type NewInput struct {
	Repo      repository.Repository
	Gemini    adapter.Gemini
	Storage   adapter.Storage
	AlertID   model.AlertID
	HistoryID *model.HistoryID // Optional: specify to continue existing conversation
}

func New(ctx context.Context, input NewInput) (*Session, error) {
	return &Session{
		repo:    input.Repo,
		gemini:  input.Gemini,
		storage: input.Storage,
		alertID: input.AlertID,
		history: &model.History{
			Contents: []*genai.Content{},
		},
	}, nil
}

func (s *Session) Send(ctx context.Context, message string) (*genai.GenerateContentResponse, error) {
	return nil, nil
}
