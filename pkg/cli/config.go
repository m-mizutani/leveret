package cli

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/leveret/pkg/adapter"
	"github.com/m-mizutani/leveret/pkg/repository"
	"github.com/urfave/cli/v3"
)

// config holds configuration values
type config struct {
	// Repository
	project  string
	database string

	// Adapters
	anthropicAPIKey string
	geminiProject   string
	geminiLocation  string
}

// globalFlags returns common flags used across commands with destination config
func globalFlags(cfg *config) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "project",
			Aliases:     []string{"p"},
			Usage:       "Google Cloud project ID",
			Sources:     cli.EnvVars("GOOGLE_CLOUD_PROJECT"),
			Destination: &cfg.project,
		},
		&cli.StringFlag{
			Name:        "database",
			Aliases:     []string{"d"},
			Usage:       "Firestore database ID",
			Value:       "(default)",
			Sources:     cli.EnvVars("FIRESTORE_DATABASE_ID"),
			Destination: &cfg.database,
		},
	}
}

// llmFlags returns flags for LLM-related configuration with destination config
func llmFlags(cfg *config) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "anthropic-api-key",
			Usage:       "Anthropic API key",
			Sources:     cli.EnvVars("ANTHROPIC_API_KEY"),
			Destination: &cfg.anthropicAPIKey,
		},
		&cli.StringFlag{
			Name:        "gemini-project",
			Usage:       "Google Cloud project ID for Gemini",
			Sources:     cli.EnvVars("GEMINI_PROJECT_ID"),
			Destination: &cfg.geminiProject,
		},
		&cli.StringFlag{
			Name:        "gemini-location",
			Usage:       "Google Cloud location for Gemini",
			Value:       "us-central1",
			Sources:     cli.EnvVars("GEMINI_LOCATION"),
			Destination: &cfg.geminiLocation,
		},
	}
}

// newRepository creates a new repository instance
func (cfg *config) newRepository() (repository.Repository, error) {
	if cfg.project == "" {
		return nil, goerr.New("project is required")
	}
	if cfg.database == "" {
		return nil, goerr.New("database is required")
	}

	repo, err := repository.New(cfg.project, cfg.database)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create repository")
	}
	return repo, nil
}

// newClaude creates a new Claude adapter instance
func (cfg *config) newClaude() (adapter.Claude, error) {
	if cfg.anthropicAPIKey == "" {
		return nil, goerr.New("anthropic-api-key is required")
	}
	return adapter.NewClaude(cfg.anthropicAPIKey), nil
}

// newGemini creates a new Gemini adapter instance
func (cfg *config) newGemini() (adapter.Gemini, error) {
	if cfg.geminiProject == "" {
		return nil, goerr.New("gemini-project is required")
	}
	if cfg.geminiLocation == "" {
		return nil, goerr.New("gemini-location is required")
	}
	return adapter.NewGemini(cfg.geminiProject, cfg.geminiLocation), nil
}

// newStorage creates a new Storage adapter instance
func (cfg *config) newStorage(ctx context.Context, bucketName string) (adapter.Storage, error) {
	if bucketName == "" {
		return nil, goerr.New("bucket name is required")
	}

	storage, err := adapter.NewStorage(ctx, bucketName)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create storage")
	}
	return storage, nil
}
