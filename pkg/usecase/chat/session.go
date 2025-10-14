package chat

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/m-mizutani/leveret/pkg/adapter"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/repository"
)

// Session manages an interactive chat session for alert analysis
type Session struct {
	repo    repository.Repository
	claude  adapter.Claude
	storage adapter.Storage
	alertID model.AlertID
	alert   *model.Alert
	history *model.History
}

// NewInput contains parameters for creating a new chat session
type NewInput struct {
	Repo      repository.Repository
	Claude    adapter.Claude
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
		claude:  input.Claude,
		storage: input.Storage,
		alertID: input.AlertID,
		history: &model.History{
			Messages: []anthropic.MessageParam{},
		},
	}, nil
}

// Send sends a message to Claude and returns the response
// 1. Add user message to history
// 2. Send messages to Claude API (Tool Call loop)
// 3. Execute external tools as requested by Claude
// 4. Add assistant response to history
// 5. Save updated conversation history to storage
func (s *Session) Send(ctx context.Context, message string) (*anthropic.Message, error) {
	// TODO: Implement Send
	// - Add user message to history.Messages
	// - Call Claude API with history.Messages
	// - Handle Tool Call loop
	// - Add assistant response to history.Messages
	// - Save conversation history (model.History) to storage
	// - Return response
	return nil, nil
}
