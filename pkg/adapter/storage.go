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
	prefix     string
	client     *storage.Client
}

// StorageOption is a functional option for configuring Storage
type StorageOption func(*storageClient)

// WithPrefix sets the prefix for object keys
func WithPrefix(prefix string) StorageOption {
	return func(s *storageClient) {
		s.prefix = prefix
	}
}

// NewStorage creates a new Cloud Storage client
func NewStorage(ctx context.Context, bucketName string, opts ...StorageOption) (Storage, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create storage client")
	}

	s := &storageClient{
		bucketName: bucketName,
		client:     client,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

func (s *storageClient) Put(ctx context.Context, key string) (io.WriteCloser, error) {
	objectKey := s.buildObjectKey(key)
	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(objectKey)
	writer := obj.NewWriter(ctx)

	return writer, nil
}

func (s *storageClient) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	objectKey := s.buildObjectKey(key)
	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(objectKey)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read from storage", goerr.Value("key", key))
	}

	return reader, nil
}

func (s *storageClient) buildObjectKey(key string) string {
	return s.prefix + key
}
