-- +goose Up
create table refresh_tokens(
    token TEXT primary key,
    created_at timestamp not null,
    updated_at timestamp not null,
    user_id varchar(36),
    expires_at timestamp,
    revoked_at timestamp,
    constraint fk_user_id
    foreign key (user_id)
    references users(id) on delete cascade
);

-- +goose Down
drop table refresh_tokens;
