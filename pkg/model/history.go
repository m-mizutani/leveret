package model

import (
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/google/uuid"
)

type HistoryID string

// NewHistoryID generates a new unique HistoryID
func NewHistoryID() HistoryID {
	return HistoryID(uuid.New().String())
}

// History represents a conversation history for alert analysis
type History struct {
	ID        HistoryID
	Title     string
	AlertID   AlertID
	CreatedAt time.Time
	UpdatedAt time.Time

	// Do not save history raw data due to size limitation of firestore
	Messages []anthropic.MessageParam `firestore:"-"`
}
