package model

import (
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
)

var (
	ErrInvalidConclusion = goerr.New("invalid conclusion")
)

type AlertID string

// NewAlertID generates a new unique AlertID
func NewAlertID() AlertID {
	return AlertID(uuid.New().String())
}

type AttributeType string

const (
	AttributeTypeString    AttributeType = "string"
	AttributeTypeNumber    AttributeType = "number"
	AttributeTypeIPAddress AttributeType = "ip_address"
	AttributeTypeDomain    AttributeType = "domain"
	AttributeTypeHash      AttributeType = "hash"
	AttributeTypeURL       AttributeType = "url"
)

type Conclusion string

const (
	ConclusionUnaffected    Conclusion = "unaffected"
	ConclusionFalsePositive Conclusion = "false_positive"
	ConclusionTruePositive  Conclusion = "true_positive"
)

// Validate checks if the conclusion is valid
func (c Conclusion) Validate() error {
	switch c {
	case ConclusionUnaffected, ConclusionFalsePositive, ConclusionTruePositive:
		return nil
	default:
		return ErrInvalidConclusion
	}
}

type Alert struct {
	ID          AlertID
	Title       string
	Description string
	Data        any
	Attributes  []*Attribute
	Embedding   firestore.Vector32

	CreatedAt  time.Time
	ResolvedAt *time.Time
	Conclusion Conclusion
	Note       string
	MergedTo   AlertID
}

type Attribute struct {
	Key   string        `json:"key"`
	Value string        `json:"value"`
	Type  AttributeType `json:"type"`
}

// Validate checks if the attribute is valid
func (a *Attribute) Validate() error {
	if a.Key == "" {
		return goerr.New("attribute key is empty")
	}
	if a.Value == "" {
		return goerr.New("attribute value is empty")
	}
	switch a.Type {
	case AttributeTypeString, AttributeTypeNumber, AttributeTypeIPAddress, AttributeTypeDomain, AttributeTypeHash, AttributeTypeURL:
		return nil
	default:
		return goerr.New("invalid attribute type", goerr.V("type", a.Type))
	}
}
