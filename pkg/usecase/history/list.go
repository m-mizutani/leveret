package history

import (
	"context"

	"github.com/m-mizutani/leveret/pkg/model"
	"github.com/m-mizutani/leveret/pkg/repository"
)

// List retrieves conversation histories from the repository
func List(
	ctx context.Context,
	repo repository.Repository,
	offset, limit int,
) ([]*model.History, error) {
	// TODO: Implement List
	return nil, nil
}
