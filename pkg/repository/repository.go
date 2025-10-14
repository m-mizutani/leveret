package repository

import (
	"context"

	"github.com/m-mizutani/leveret/pkg/model"
)

// Repository defines the interface for alert data persistence
type Repository interface {
	// PutAlert saves an alert to the repository
	PutAlert(ctx context.Context, alert *model.Alert) error

	// GetAlert retrieves an alert by ID
	GetAlert(ctx context.Context, id model.AlertID) (*model.Alert, error)

	// ListAlerts retrieves alerts with optional filters
	ListAlerts(ctx context.Context, offset, limit int) ([]*model.Alert, error)

	// SearchSimilarAlerts performs vector search to find similar alerts
	SearchSimilarAlerts(ctx context.Context, embedding []float64, limit int) ([]*model.Alert, error)

	// PutHistory saves a conversation history to the repository
	PutHistory(ctx context.Context, history *model.History) error

	// GetHistory retrieves a conversation history by ID
	GetHistory(ctx context.Context, id model.HistoryID) (*model.History, error)

	// ListHistory retrieves conversation histories
	ListHistory(ctx context.Context, offset, limit int) ([]*model.History, error)
}
