package alert

import (
	"context"
	"time"

	"github.com/m-mizutani/leveret/pkg/model"
)

func (u *UseCase) Insert(
	ctx context.Context,
	data any,
) (*model.Alert, error) {
	alert := &model.Alert{
		ID:        model.NewAlertID(),
		Data:      data,
		CreatedAt: time.Now(),
	}

	if err := u.repo.PutAlert(ctx, alert); err != nil {
		return nil, err
	}

	return alert, nil
}
