package chat

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/adapter"
	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/repository"
	"google.golang.org/genai"
)

// loadHistory loads conversation history from Cloud Storage and repository
func loadHistory(ctx context.Context, repo repository.Repository, storage adapter.Storage, historyID model.HistoryID) (*model.History, error) {
	// Get history metadata from repository
	history, err := repo.GetHistory(ctx, historyID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get history from repository")
	}

	// Load conversation contents from Cloud Storage
	reader, err := storage.Get(ctx, "histories/"+string(historyID)+".json")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get history from storage")
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read history data")
	}

	var contents []*genai.Content
	if err := json.Unmarshal(data, &contents); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal history contents")
	}

	history.Contents = contents
	return history, nil
}

// saveHistory saves conversation history to Cloud Storage and repository
func saveHistory(ctx context.Context, repo repository.Repository, storage adapter.Storage, alertID model.AlertID, history *model.History) error {
	// Generate new history ID if not exists
	if history.ID == "" {
		history.ID = model.NewHistoryID()
		history.AlertID = alertID
		history.CreatedAt = time.Now()
	}
	history.UpdatedAt = time.Now()

	// Save contents to Cloud Storage
	writer, err := storage.Put(ctx, "histories/"+string(history.ID)+".json")
	if err != nil {
		return goerr.Wrap(err, "failed to create storage writer")
	}
	defer writer.Close()

	data, err := json.Marshal(history.Contents)
	if err != nil {
		return goerr.Wrap(err, "failed to marshal history contents")
	}

	if _, err := writer.Write(data); err != nil {
		return goerr.Wrap(err, "failed to write history to storage")
	}

	if err := writer.Close(); err != nil {
		return goerr.Wrap(err, "failed to close storage writer")
	}

	// Save metadata to repository
	if err := repo.PutHistory(ctx, history); err != nil {
		return goerr.Wrap(err, "failed to put history to repository")
	}

	return nil
}

// generateTitle generates a short title from the first user message
func generateTitle(ctx context.Context, gemini adapter.Gemini, message string) (string, error) {
	prompt := "Generate a short title (max 50 characters) that summarizes the following question or topic. Return only the title, nothing else:\n\n" + message

	contents := []*genai.Content{
		genai.NewContentFromText(prompt, genai.RoleUser),
	}

	resp, err := gemini.GenerateContent(ctx, contents, nil)
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate title")
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return "", goerr.New("no response from LLM")
	}

	var title strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Text != "" {
			title.WriteString(part.Text)
		}
	}

	// Trim whitespace and limit length
	return strings.TrimSpace(title.String()), nil
}
