package alert

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/model"
)

// Unmerge unmerges a previously merged alert
func (u *UseCase) Unmerge(
	ctx context.Context,
	alertID model.AlertID,
) error {
	alert, err := u.repo.GetAlert(ctx, alertID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert", goerr.Value("alertID", alertID))
	}

	// Clear the merge reference
	alert.MergedTo = ""

	if err := u.repo.PutAlert(ctx, alert); err != nil {
		return goerr.Wrap(err, "failed to update alert", goerr.Value("alertID", alertID))
	}

	return nil
}
