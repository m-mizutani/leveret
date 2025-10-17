package history

import (
	"context"

	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/repository"
)

func List(
	ctx context.Context,
	repo repository.Repository,
	offset, limit int,
) ([]*model.History, error) {
	return nil, nil
}
