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
	claudeAPIKey   string
	geminiProject  string
	geminiLocation string

	// Storage
	bucketName string
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
		&cli.StringFlag{
			Name:        "bucket",
			Aliases:     []string{"b"},
			Usage:       "Cloud Storage bucket name",
			Sources:     cli.EnvVars("STORAGE_BUCKET_NAME"),
			Destination: &cfg.bucketName,
		},
	}
}

// llmFlags returns flags for LLM-related configuration with destination config
func llmFlags(cfg *config) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "claude-api-key",
			Usage:       "Claude API key",
			Sources:     cli.EnvVars("CLAUDE_API_KEY"),
			Destination: &cfg.claudeAPIKey,
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
	if cfg.claudeAPIKey == "" {
		return nil, goerr.New("claude-api-key is required")
	}
	return adapter.NewClaude(cfg.claudeAPIKey), nil
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
func (cfg *config) newStorage(ctx context.Context) (adapter.Storage, error) {
	if cfg.bucketName == "" {
		return nil, goerr.New("bucket name is required")
	}

	storage, err := adapter.NewStorage(ctx, cfg.bucketName)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create storage")
	}
	return storage, nil
}
