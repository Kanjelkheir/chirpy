-- name: CreateUser :one
insert into users (id, created_at, updated_at, email, password) values(
    $1,
    $2,
    $3,
    $4,
    $5
) returning *;

-- name: DeleteUser :exec
delete from users where email = $1;

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
