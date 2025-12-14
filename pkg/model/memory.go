package model

import (
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
)

type MemoryID string

// NewMemoryID generates a new unique MemoryID
func NewMemoryID() MemoryID {
	return MemoryID(uuid.New().String())
}

// Memory represents a stored knowledge claim from a BigQuery sub-agent session
type Memory struct {
	ID        MemoryID
	Claim     string
	QueryText string
	Embedding firestore.Vector32
	Score     float64
	CreatedAt time.Time
	UpdatedAt time.Time
}
