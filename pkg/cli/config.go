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
	firestoreProject string
	database         string

	// Adapters
	geminiProject         string
	geminiLocation        string
	geminiGenerativeModel string
	geminiEmbeddingModel  string

	// Storage
	bucketName    string
	storagePrefix string
}

// globalFlags returns common flags used across commands with destination config
func globalFlags(cfg *config) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "firestore-project",
			Aliases:     []string{"p"},
			Usage:       "Google Cloud project ID for Firestore",
			Sources:     cli.EnvVars("LEVERET_FIRESTORE_PROJECT"),
			Destination: &cfg.firestoreProject,
		},
		&cli.StringFlag{
			Name:        "firestore-database",
			Aliases:     []string{"d"},
			Usage:       "Firestore database ID",
			Sources:     cli.EnvVars("LEVERET_FIRESTORE_DATABASE_ID"),
			Destination: &cfg.database,
		},
		&cli.StringFlag{
			Name:        "storage-bucket",
			Aliases:     []string{"b"},
			Usage:       "Cloud Storage bucket name",
			Sources:     cli.EnvVars("LEVERET_STORAGE_BUCKET"),
			Destination: &cfg.bucketName,
		},
		&cli.StringFlag{
			Name:        "storage-prefix",
			Usage:       "Cloud Storage object key prefix",
			Sources:     cli.EnvVars("LEVERET_STORAGE_PREFIX"),
			Destination: &cfg.storagePrefix,
		},
	}
}

// llmFlags returns flags for LLM-related configuration with destination config
func llmFlags(cfg *config) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "gemini-project",
			Usage:       "Google Cloud project ID for Gemini API",
			Sources:     cli.EnvVars("LEVERET_GEMINI_PROJECT"),
			Destination: &cfg.geminiProject,
		},
		&cli.StringFlag{
			Name:        "gemini-location",
			Usage:       "Google Cloud location for Gemini API",
			Value:       "us-central1",
			Sources:     cli.EnvVars("LEVERET_GEMINI_LOCATION"),
			Destination: &cfg.geminiLocation,
		},
		&cli.StringFlag{
			Name:        "gemini-generative-model",
			Usage:       "Gemini generative model name",
			Value:       "gemini-2.5-flash",
			Sources:     cli.EnvVars("LEVERET_GEMINI_GENERATIVE_MODEL"),
			Destination: &cfg.geminiGenerativeModel,
		},
		&cli.StringFlag{
			Name:        "gemini-embedding-model",
			Usage:       "Gemini embedding model name",
			Value:       "gemini-embedding-001",
			Sources:     cli.EnvVars("LEVERET_GEMINI_EMBEDDING_MODEL"),
			Destination: &cfg.geminiEmbeddingModel,
		},
	}
}

// newRepository creates a new repository instance
func (cfg *config) newRepository() (repository.Repository, error) {
	if cfg.firestoreProject == "" {
		return nil, goerr.New("firestore-project is required")
	}
	if cfg.database == "" {
		return nil, goerr.New("database is required")
	}

	repo, err := repository.New(cfg.firestoreProject, cfg.database)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create repository")
	}
	return repo, nil
}

// newGemini creates a new Gemini adapter instance
func (cfg *config) newGemini(ctx context.Context) (adapter.Gemini, error) {
	if cfg.geminiProject == "" {
		return nil, goerr.New("gemini-project is required")
	}

	var opts []adapter.GeminiOption
	if cfg.geminiGenerativeModel != "" {
		opts = append(opts, adapter.WithGenerativeModel(cfg.geminiGenerativeModel))
	}
	if cfg.geminiEmbeddingModel != "" {
		opts = append(opts, adapter.WithEmbeddingModel(cfg.geminiEmbeddingModel))
	}

	return adapter.NewGemini(ctx, cfg.geminiProject, cfg.geminiLocation, opts...)
}

// newStorage creates a new Storage adapter instance
func (cfg *config) newStorage(ctx context.Context) (adapter.Storage, error) {
	if cfg.bucketName == "" {
		return nil, goerr.New("bucket name is required")
	}

	var opts []adapter.StorageOption
	if cfg.storagePrefix != "" {
		opts = append(opts, adapter.WithPrefix(cfg.storagePrefix))
	}

	storage, err := adapter.NewStorage(ctx, cfg.bucketName, opts...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create storage")
	}
	return storage, nil
}
