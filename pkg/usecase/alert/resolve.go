package alert

import (
	"context"
	"time"

	"github.com/m-mizutani/leveret/pkg/model"
)

// Resolve marks an alert as resolved
func (u *UseCase) Resolve(
	ctx context.Context,
	alertID model.AlertID,
	conclusion model.Conclusion,
	note string,
) error {
	if err := conclusion.Validate(); err != nil {
		return err
	}

	alert, err := u.repo.GetAlert(ctx, alertID)
	if err != nil {
		return err
	}

	now := time.Now()
	alert.ResolvedAt = &now
	alert.Conclusion = conclusion
	alert.Note = note

	if err := u.repo.PutAlert(ctx, alert); err != nil {
		return err
	}

	return nil
}
