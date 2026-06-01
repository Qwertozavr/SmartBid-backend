-- +goose Up
-- +goose StatementBegin
alter table ads
add column if not exists photo bytea;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table ads
drop column if exists photo;
-- +goose StatementEnd
