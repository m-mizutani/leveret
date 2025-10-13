package usecase

import "github.com/m-mizutani/leveret/pkg/interfaces"

type UseCases struct{}

func New(repo interfaces.Repository) *UseCases {
	return &UseCases{}
}
