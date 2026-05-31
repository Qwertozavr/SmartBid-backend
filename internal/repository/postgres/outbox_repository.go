package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"smartbid-backend/internal/domain"
)

type OutboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

func (r *OutboxRepository) Create(ctx context.Context, event domain.OutboxEvent) error {
	const query = `
		insert into outbox_events (
			id,
			topic,
			event_type,
			aggregate_type,
			aggregate_id,
			payload,
			status
		)
		values ($1, $2, $3, $4, $5, $6, $7)
		on conflict (aggregate_type, aggregate_id, event_type) do nothing
	`

	_, err := executor(ctx, r.pool).Exec(
		ctx,
		query,
		event.ID,
		event.Topic,
		event.EventType,
		event.AggregateType,
		event.AggregateID,
		event.Payload,
		domain.OutboxEventStatusPending,
	)
	return err
}

func (r *OutboxRepository) ClaimPending(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	const query = `
		with claimed as (
			select id
			from outbox_events
			where status in ($1, $2)
				or (status = $3 and updated_at < now() - interval '1 minute')
			order by created_at
			limit $4
			for update skip locked
		)
		update outbox_events
		set status = $5,
			attempts = attempts + 1,
			updated_at = now()
		from claimed
		where outbox_events.id = claimed.id
		returning
			outbox_events.id::text,
			outbox_events.topic,
			outbox_events.event_type,
			outbox_events.aggregate_type,
			outbox_events.aggregate_id,
			outbox_events.payload::text,
			outbox_events.status,
			outbox_events.attempts,
			coalesce(outbox_events.last_error, ''),
			outbox_events.created_at,
			outbox_events.updated_at
	`

	rows, err := executor(ctx, r.pool).Query(
		ctx,
		query,
		domain.OutboxEventStatusPending,
		domain.OutboxEventStatusFailed,
		domain.OutboxEventStatusProcessing,
		limit,
		domain.OutboxEventStatusProcessing,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]domain.OutboxEvent, 0, limit)
	for rows.Next() {
		var event domain.OutboxEvent
		var payload string
		err := rows.Scan(
			&event.ID,
			&event.Topic,
			&event.EventType,
			&event.AggregateType,
			&event.AggregateID,
			&payload,
			&event.Status,
			&event.Attempts,
			&event.LastError,
			&event.CreatedAt,
			&event.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		event.Payload = []byte(payload)
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, id string) error {
	const query = `
		update outbox_events
		set status = $1,
			last_error = null,
			updated_at = now()
		where id = $2
	`

	commandTag, err := executor(ctx, r.pool).Exec(ctx, query, domain.OutboxEventStatusPublished, id)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, id string, cause error, maxAttempts int) error {
	if cause == nil {
		cause = errors.New("unknown outbox publish error")
	}

	status := domain.OutboxEventStatusFailed
	if maxAttempts > 0 {
		const attemptsQuery = `select attempts from outbox_events where id = $1`
		var attempts int
		if err := executor(ctx, r.pool).QueryRow(ctx, attemptsQuery, id).Scan(&attempts); err != nil {
			return err
		}
		if attempts >= maxAttempts {
			status = domain.OutboxEventStatusDead
		}
	}

	const query = `
		update outbox_events
		set status = $1,
			last_error = $2,
			updated_at = now()
		where id = $3
	`

	commandTag, err := executor(ctx, r.pool).Exec(ctx, query, status, fmt.Sprintf("%v", cause), id)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}
