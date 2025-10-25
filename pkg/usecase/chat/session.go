package chat

import (
	"context"
	"encoding/json"

	"github.com/m-mizutani/goerr/v2"
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
	alert, err := input.Repo.GetAlert(ctx, input.AlertID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get alert")
	}

	return &Session{
		repo:    input.Repo,
		gemini:  input.Gemini,
		storage: input.Storage,

		alertID: input.AlertID,
		alert:   alert,
		history: &model.History{},
	}, nil
}

func (s *Session) Send(ctx context.Context, message string) (*genai.GenerateContentResponse, error) {
	// Build system prompt with alert data
	alertData, err := json.MarshalIndent(s.alert.Data, "", "  ")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal alert data")
	}

	systemPrompt := "You are a helpful assistant. When asked about the alert, refer to the following data:\n\nAlert Data:\n" + string(alertData)

	// Add user message to history
	userContent := genai.NewContentFromText(message, genai.RoleUser)
	s.history.Contents = append(s.history.Contents, userContent)

	// Generate response
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, ""),
	}

	resp, err := s.gemini.GenerateContent(ctx, s.history.Contents, config)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate content")
	}

	// Add assistant response to history
	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		s.history.Contents = append(s.history.Contents, resp.Candidates[0].Content)
	}

	return resp, nil
}
