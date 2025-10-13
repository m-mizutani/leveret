package repository

import (
	"context"

	"github.com/m-mizutani/leveret/pkg/interfaces"
	"github.com/m-mizutani/leveret/pkg/model"
)

// firestoreRepo implements Repository interface using Firestore
type firestoreRepo struct {
	projectID    string
	databaseName string
}

// NewFirestore creates a new Firestore repository
func NewFirestore(projectID, databaseName string) (interfaces.Repository, error) {
	return &firestoreRepo{
		projectID:    projectID,
		databaseName: databaseName,
	}, nil
}

func (r *firestoreRepo) SaveAlert(ctx context.Context, alert *model.Alert) error {
	// TODO: Implement Firestore integration
	return nil
}

func (r *firestoreRepo) GetAlert(ctx context.Context, id model.AlertID) (*model.Alert, error) {
	// TODO: Implement Firestore integration
	return nil, nil
}

func (r *firestoreRepo) ListAlerts(ctx context.Context, filters ...interfaces.Filter) ([]*model.Alert, error) {
	// TODO: Implement Firestore integration
	return nil, nil
}

func (r *firestoreRepo) UpdateAlert(ctx context.Context, alert *model.Alert) error {
	// TODO: Implement Firestore integration
	return nil
}

func (r *firestoreRepo) SearchSimilarAlerts(ctx context.Context, embedding []float64, limit int) ([]*model.Alert, error) {
	// TODO: Implement Firestore vector search
	return nil, nil
}
