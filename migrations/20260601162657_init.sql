-- +goose Up
-- +goose StatementBegin
create extension if not exists pgcrypto;

create table if not exists ad_status (
    id uuid primary key default gen_random_uuid(),
    slug text not null unique,
    description text not null
);

insert into ad_status (slug, description)
values
    ('created', 'Объявление создано'),
    ('published', 'Объявление опубликовано'),
    ('canceled', 'Объявление отменено'),
    ('bought', 'Товар куплен'),
    ('removed', 'Объявление удалено')
on conflict (slug) do update
set description = excluded.description;

create table if not exists ads (
    id uuid primary key default gen_random_uuid(),
    title text not null,
    description text not null,
    start_price bigint not null check (start_price >= 0),
    final_price bigint not null check (final_price >= 0),
    status_id uuid not null references ad_status(id),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists ads;
drop table if exists ad_status;
-- +goose StatementEnd
