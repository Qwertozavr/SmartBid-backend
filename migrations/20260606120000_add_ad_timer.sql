-- +goose Up
-- +goose StatementBegin
insert into ad_status (slug, description)
values ('expired', 'Срок объявления истёк')
on conflict (slug) do update
set description = excluded.description;

alter table ads
add column if not exists published_at timestamptz;

alter table ads
add column if not exists expires_at timestamptz;

create index if not exists ads_published_expires_at_idx
on ads (expires_at)
where expires_at is not null;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop index if exists ads_published_expires_at_idx;

alter table ads
drop column if exists expires_at;

alter table ads
drop column if exists published_at;

delete from ad_status
where slug = 'expired';
-- +goose StatementEnd
