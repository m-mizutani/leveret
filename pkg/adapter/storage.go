package adapter

import (
	"context"
	"io"

	"cloud.google.com/go/storage"
	"github.com/m-mizutani/goerr/v2"
)

// Storage is the interface for conversation history storage
type Storage interface {
	// Put returns a writer to save conversation history to storage
	Put(ctx context.Context, key string) (io.WriteCloser, error)
	// Get loads conversation history from storage
	Get(ctx context.Context, key string) (io.ReadCloser, error)
}

// storageClient implements Storage interface using Cloud Storage
type storageClient struct {
	bucketName string
	client     *storage.Client
}

// NewStorage creates a new Cloud Storage client
func NewStorage(ctx context.Context, bucketName string) (Storage, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create storage client")
	}

	return &storageClient{
		bucketName: bucketName,
		client:     client,
	}, nil
}

func (s *storageClient) Put(ctx context.Context, key string) (io.WriteCloser, error) {
	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(key)
	writer := obj.NewWriter(ctx)
	return writer, nil
}

func (s *storageClient) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(key)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read from storage", goerr.Value("key", key))
	}

	return reader, nil
}
