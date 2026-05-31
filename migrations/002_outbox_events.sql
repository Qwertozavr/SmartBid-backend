-- +goose Up
-- +goose StatementBegin
create table if not exists outbox_events (
    id uuid primary key default gen_random_uuid(),
    topic text not null,
    event_type text not null,
    aggregate_type text not null,
    aggregate_id text not null,
    payload jsonb not null,
    status text not null default 'pending'
        check (status in ('pending', 'processing', 'published', 'failed', 'dead')),
    attempts integer not null default 0 check (attempts >= 0),
    last_error text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (aggregate_type, aggregate_id, event_type)
);

create index if not exists outbox_events_status_created_at_idx
on outbox_events (status, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists outbox_events;
-- +goose StatementEnd
