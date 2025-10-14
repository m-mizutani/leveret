package alert

import (
	"context"

	"github.com/m-mizutani/leveret/pkg/model"
)

// Show retrieves and displays detailed information of a specific alert
// 1. Fetch alert from Firestore by ID
// 2. Return the alert with all details (attributes, metadata, etc.)
func (u *UseCase) Show(
	ctx context.Context,
	alertID model.AlertID,
) (*model.Alert, error) {
	alert, err := u.repo.GetAlert(ctx, alertID)
	if err != nil {
		return nil, err
	}

	return alert, nil
}
