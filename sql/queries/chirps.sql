-- name: CreateChirp :one
insert into chirps (id, created_at, updated_at, body, user_id) values (
    $1,
    $2,
    $3,
    $4,
    $5
) returning *;


-- name: GetChirpsAsc :many
select * from chirps order by created_at asc;

-- name: GetChirpsDesc :many
select * from chirps order by created_at desc;

-- name: GetChirp :one
select * from chirps where id = $1;


-- name: DeleteChirp :exec
delete from chirps where user_id = $1;

-- name: GetChirpByAuthorAsc :many
select * from chirps where user_id = $1 order by created_at asc;

-- name: GetChirpByAuthorDesc :many
select * from chirps where user_id = $1 order by created_at desc;
