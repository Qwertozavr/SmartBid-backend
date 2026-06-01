-- +goose Up
-- +goose StatementBegin
alter table ads
add column if not exists pretendent_id integer;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table ads
drop column if exists pretendent_id;
-- +goose StatementEnd
