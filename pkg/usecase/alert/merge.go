package alert

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/model"
)

// Merge consolidates source alert into target alert
func (u *UseCase) Merge(
	ctx context.Context,
	sourceID, targetID model.AlertID,
) error {
	// Get both alerts
	source, err := u.repo.GetAlert(ctx, sourceID)
	if err != nil {
		return goerr.Wrap(err, "failed to get source alert", goerr.Value("sourceID", sourceID))
	}

	_, err = u.repo.GetAlert(ctx, targetID)
	if err != nil {
		return goerr.Wrap(err, "failed to get target alert", goerr.Value("targetID", targetID))
	}

	// Mark source as merged to target
	source.MergedTo = targetID

	if err := u.repo.PutAlert(ctx, source); err != nil {
		return goerr.Wrap(err, "failed to update source alert", goerr.Value("sourceID", sourceID))
	}

	return nil
}
