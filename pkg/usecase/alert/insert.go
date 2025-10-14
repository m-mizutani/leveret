package alert

import (
	"context"
	"time"

	"github.com/m-mizutani/leveret/pkg/model"
)

// Insert registers a new alert
// 1. Parse JSON alert data
// 2. Run policy evaluation (accept/reject)
// 3. Generate summary and extract IOCs via Claude API
// 4. Generate embedding vector via Gemini API
// 5. Save to Firestore with generated alert ID
func (u *UseCase) Insert(
	ctx context.Context,
	data any,
) (*model.Alert, error) {
	alert := &model.Alert{
		ID:        model.NewAlertID(),
		Data:      data,
		CreatedAt: time.Now(),
	}

	// TODO: Run policy evaluation (accept/reject)

	// TODO: Generate summary and extract IOCs via Claude API
	// This should populate:
	// - alert.Title
	// - alert.Description
	// - alert.Attributes

	// TODO: Generate embedding vector via Gemini API

	// Save to Firestore
	if err := u.repo.PutAlert(ctx, alert); err != nil {
		return nil, err
	}

	return alert, nil
}
