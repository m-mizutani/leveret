package alert

import (
	"io"
	"os"

	"github.com/m-mizutani/leveret/pkg/adapter"
	"github.com/m-mizutani/leveret/pkg/repository"
)

// UseCase provides alert-related operations
type UseCase struct {
	repo   repository.Repository
	gemini adapter.Gemini
	output io.Writer
}

// Option is a functional option for UseCase
type Option func(*UseCase)

// WithOutput sets the output writer
func WithOutput(w io.Writer) Option {
	return func(uc *UseCase) {
		uc.output = w
	}
}

// New creates a new alert UseCase instance
func New(
	repo repository.Repository,
	gemini adapter.Gemini,
	opts ...Option,
) *UseCase {
	uc := &UseCase{
		repo:   repo,
		gemini: gemini,
		output: os.Stdout,
	}

	for _, opt := range opts {
		opt(uc)
	}

	return uc
}
