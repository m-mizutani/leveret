package adapter

import (
	"context"
	"io"
)

// Storage is the interface for conversation history storage
type Storage interface {
	// Save saves conversation history to storage
	Save(ctx context.Context, key string, data io.Reader) error
	// Load loads conversation history from storage
	Load(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete deletes conversation history from storage
	Delete(ctx context.Context, key string) error
}

// storageClient implements Storage interface using Cloud Storage
type storageClient struct {
	bucketName string
}

// NewStorage creates a new Cloud Storage client
func NewStorage(bucketName string) Storage {
	return &storageClient{
		bucketName: bucketName,
	}
}

func (s *storageClient) Save(ctx context.Context, key string, data io.Reader) error {
	// TODO: Implement actual Cloud Storage integration
	return nil
}

func (s *storageClient) Load(ctx context.Context, key string) (io.ReadCloser, error) {
	// TODO: Implement actual Cloud Storage integration
	return nil, nil
}

func (s *storageClient) Delete(ctx context.Context, key string) error {
	// TODO: Implement actual Cloud Storage integration
	return nil
}
