-- +goose Up
create table users(
    id varchar(36) primary key,
    created_at timestamp not null,
    updated_at timestamp not null,
    email varchar(255) not null unique
);

-- +goose Down
drop table users;
