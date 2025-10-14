package alert

import (
	"context"

	"github.com/m-mizutani/leveret/pkg/model"
)

// ListOptions contains options for listing alerts
type ListOptions struct {
	IncludeMerged bool
	Offset        int
	Limit         int
}

// List retrieves a list of alerts
func (u *UseCase) List(
	ctx context.Context,
	opts ListOptions,
) ([]*model.Alert, error) {
	alerts, err := u.repo.ListAlerts(ctx, opts.Offset, opts.Limit)
	if err != nil {
		return nil, err
	}

	// Filter out merged alerts if not included
	if !opts.IncludeMerged {
		filtered := make([]*model.Alert, 0, len(alerts))
		for _, alert := range alerts {
			if alert.MergedTo == "" {
				filtered = append(filtered, alert)
			}
		}
		return filtered, nil
	}

	return alerts, nil
}
