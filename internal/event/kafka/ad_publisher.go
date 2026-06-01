package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"

	"smartbid-backend/internal/domain"
)

type Publisher struct {
	writer *kafka.Writer
	dlq    *kafka.Writer
}

type deadLetterMessage struct {
	EventID       string          `json:"event_id"`
	OriginalTopic string          `json:"original_topic"`
	EventType     string          `json:"event_type"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	Payload       json.RawMessage `json:"payload"`
	Error         string          `json:"error"`
}

func NewPublisher(brokers []string, dlqTopic string) *Publisher {
	return &Publisher{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireAll,
		},
		dlq: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        dlqTopic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireAll,
		},
	}
}

func (p *Publisher) Publish(ctx context.Context, event domain.OutboxEvent) error {
	err := p.writer.WriteMessages(ctx, kafka.Message{
		Topic: event.Topic,
		Key:   []byte(event.ID),
		Value: event.Payload,
		Headers: []kafka.Header{
			{Key: "event_id", Value: []byte(event.ID)},
			{Key: "event_type", Value: []byte(event.EventType)},
			{Key: "aggregate_type", Value: []byte(event.AggregateType)},
			{Key: "aggregate_id", Value: []byte(event.AggregateID)},
		},
	})
	if err != nil {
		return fmt.Errorf("publish outbox event: %w", err)
	}

	return nil
}

func (p *Publisher) PublishDeadLetter(ctx context.Context, event domain.OutboxEvent, cause error) error {
	message := deadLetterMessage{
		EventID:       event.ID,
		OriginalTopic: event.Topic,
		EventType:     event.EventType,
		AggregateType: event.AggregateType,
		AggregateID:   event.AggregateID,
		Payload:       json.RawMessage(event.Payload),
		Error:         fmt.Sprintf("%v", cause),
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal dead letter message: %w", err)
	}

	err = p.dlq.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.ID),
		Value: payload,
		Headers: []kafka.Header{
			{Key: "event_id", Value: []byte(event.ID)},
			{Key: "event_type", Value: []byte(event.EventType)},
			{Key: "dead_letter", Value: []byte("true")},
		},
	})
	if err != nil {
		return fmt.Errorf("publish dead letter event: %w", err)
	}

	return nil
}

func (p *Publisher) Close() error {
	writerErr := p.writer.Close()
	dlqErr := p.dlq.Close()
	if writerErr != nil {
		return writerErr
	}
	return dlqErr
}
