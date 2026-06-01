package domain

const (
	AdCreatedEventType = "ad.created"
	AdCreatedTopic     = "ad-created"
	AdCreatedDLQTopic  = "ad-created-dlq"
)

type AdCreatedEvent struct {
	EventID string `json:"event_id"`
	AdID    string `json:"ad_id"`
}
