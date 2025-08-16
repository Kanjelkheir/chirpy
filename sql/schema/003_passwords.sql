-- +goose Up
alter table users add column Password text not null default 'UNSET';

-- +goose Down
alter table users drop column Password;
