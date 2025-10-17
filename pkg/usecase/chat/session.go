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

// New creates a new chat session for an alert
// 1. Fetch alert from Firestore
// 2. Load conversation history from Cloud Storage (if HistoryID is specified)
// 3. Create new history if not specified
func New(ctx context.Context, input NewInput) (*Session, error) {
	// TODO: Implement New
	// - Fetch alert from repository
	// - If HistoryID is specified, load history from storage
	// - Otherwise, create new history
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

// Send sends a message to Gemini and returns the response
// 1. Add user message to history
// 2. Send messages to Gemini API (Tool Call loop)
// 3. Execute external tools as requested by Gemini
// 4. Add assistant response to history
// 5. Save updated conversation history to storage
func (s *Session) Send(ctx context.Context, message string) (*genai.GenerateContentResponse, error) {
	// TODO: Implement Send
	// - Add user message to history.Contents
	// - Call Gemini API with history.Contents
	// - Handle Tool Call loop
	// - Add assistant response to history.Contents
	// - Save conversation history (model.History) to storage
	// - Return response
	return nil, nil
}
