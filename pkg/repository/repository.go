package repository

import (
	"context"

	"github.com/m-mizutani/leveret/pkg/model"
)

// SearchAlertsInput contains parameters for searching alerts
type SearchAlertsInput struct {
	Field    string // Field path within Data (auto-prefixed with "Data.")
	Operator string // Firestore operator
	Value    any    // Value to compare
	Limit    int    // Max results (default: 10, max: 100)
	Offset   int    // Skip count for pagination
}

// Repository defines the interface for alert data persistence
type Repository interface {
	// PutAlert saves an alert to the repository
	PutAlert(ctx context.Context, alert *model.Alert) error

	// GetAlert retrieves an alert by ID
	GetAlert(ctx context.Context, id model.AlertID) (*model.Alert, error)

	// ListAlerts retrieves alerts with optional filters
	ListAlerts(ctx context.Context, offset, limit int) ([]*model.Alert, error)

	// SearchAlerts searches alerts by field conditions in Data
	SearchAlerts(ctx context.Context, input *SearchAlertsInput) ([]*model.Alert, error)

	// SearchSimilarAlerts performs vector search to find similar alerts
	SearchSimilarAlerts(ctx context.Context, embedding []float32, threshold float64) ([]*model.Alert, error)

	// PutHistory saves a conversation history to the repository
	PutHistory(ctx context.Context, history *model.History) error

	// GetHistory retrieves a conversation history by ID
	GetHistory(ctx context.Context, id model.HistoryID) (*model.History, error)

	// ListHistory retrieves conversation histories
	ListHistory(ctx context.Context, offset, limit int) ([]*model.History, error)

	// ListHistoryByAlert retrieves conversation histories for a specific alert
	ListHistoryByAlert(ctx context.Context, alertID model.AlertID) ([]*model.History, error)
}
