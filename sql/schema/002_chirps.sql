-- +goose Up
create table chirps(
    id varchar(36) primary key,
    created_at timestamp not null,
    updated_at timestamp not null,
    body text not null,
    user_id varchar(36),
    constraint fk_user_id
    foreign key (user_id)
    references users(id) on delete cascade
);


-- +goose Down
delete from chirps;
