package model

import "time"

type AlertID string

type AttributeType string

const (
	AttributeTypeString    AttributeType = "string"
	AttributeTypeNumber    AttributeType = "number"
	AttributeTypeIPAddress AttributeType = "ip_address"
	AttributeTypeDomain    AttributeType = "domain"
	AttributeTypeHash      AttributeType = "hash"
	AttributeTypeURL       AttributeType = "url"
)

type Alert struct {
	ID          AlertID
	Title       string
	Description string
	Data        any
	Attributes  []*Attribute

	CreatedAt  time.Time
	ResolvedAt *time.Time
	Conclusion string
	Note       string
	MergedTo   AlertID
}

type Attribute struct {
	Key   string
	Value string
	Type  AttributeType
}
