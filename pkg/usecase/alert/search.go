package alert

import (
	"context"

	"github.com/m-mizutani/leveret/pkg/model"
)

// SearchOptions contains options for searching similar alerts
type SearchOptions struct {
	Query string // Natural language query or alert description
	Limit int    // Maximum number of similar alerts to return
}

// Search searches for similar alerts using vector similarity
// 1. Generate embedding vector from query text via Gemini API
// 2. Perform vector search in Firestore
// 3. Return similar alerts ordered by similarity
func (u *UseCase) Search(
	ctx context.Context,
	opts SearchOptions,
) ([]*model.Alert, error) {
	// TODO: Generate embedding vector from query text via Gemini API
	// vector, err := u.gemini.GenerateEmbedding(ctx, opts.Query)

	// TODO: Perform vector search in Firestore
	// alerts, err := u.repo.SearchSimilarAlerts(ctx, vector, opts.Limit)

	return nil, nil
}
