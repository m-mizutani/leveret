package interfaces

import (
	"context"

	"github.com/m-mizutani/leveret/pkg/model"
)

// Filter is a function to filter alerts in list operations
type Filter func(*model.Alert) bool

// Repository defines the interface for alert data persistence
type Repository interface {
	// SaveAlert saves an alert to the repository
	SaveAlert(ctx context.Context, alert *model.Alert) error

	// GetAlert retrieves an alert by ID
	GetAlert(ctx context.Context, id model.AlertID) (*model.Alert, error)

	// ListAlerts retrieves alerts with optional filters
	ListAlerts(ctx context.Context, filters ...Filter) ([]*model.Alert, error)

	// UpdateAlert updates an existing alert
	UpdateAlert(ctx context.Context, alert *model.Alert) error

	// SearchSimilarAlerts performs vector search to find similar alerts
	SearchSimilarAlerts(ctx context.Context, embedding []float64, limit int) ([]*model.Alert, error)
}
