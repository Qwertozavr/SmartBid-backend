package domain

import "time"

type OutboxEventStatus string

const (
	OutboxEventStatusPending    OutboxEventStatus = "pending"
	OutboxEventStatusProcessing OutboxEventStatus = "processing"
	OutboxEventStatusPublished  OutboxEventStatus = "published"
	OutboxEventStatusFailed     OutboxEventStatus = "failed"
	OutboxEventStatusDead       OutboxEventStatus = "dead"
)

type OutboxEvent struct {
	ID            string
	Topic         string
	EventType     string
	AggregateType string
	AggregateID   string
	Payload       []byte
	Status        OutboxEventStatus
	Attempts      int
	LastError     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
