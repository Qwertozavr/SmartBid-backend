package domain

const (
	AdCreatedEventType  = "ad.created"
	AdCreatedTopic      = "ad-created"
	AdCreatedDLQTopic   = "ad-created-dlq"
	AdFinishedEventType = "ad.finished"
	AdFinishedTopic     = "ad-finished"
)

type AdCreatedEvent struct {
	EventID string `json:"event_id"`
	AdID    string `json:"ad_id"`
}

type AdFinishedEvent struct {
	EventID      string   `json:"event_id"`
	AdID         string   `json:"ad_id"`
	Status       AdStatus `json:"status"`
	PretendentID *int     `json:"pretendent_id"`
	FinalPrice   int64    `json:"final_price"`
}
