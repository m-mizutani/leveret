package model

import (
	"time"

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

	CreatedAt  time.Time
	ResolvedAt *time.Time
	Conclusion Conclusion
	Note       string
	MergedTo   AlertID
}

type Attribute struct {
	Key   string
	Value string
	Type  AttributeType
}
