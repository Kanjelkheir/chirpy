-- name: CreateChirp :one
insert into chirps (id, created_at, updated_at, body, user_id) values (
    $1,
    $2,
    $3,
    $4,
    $5
) returning *;


-- name: GetChirps :many
select * from chirps order by created_at asc;

-- name: GetChirp :one
select * from chirps where id = $1;


-- name: DeleteChirp :exec
delete from chirps where user_id = $1;
