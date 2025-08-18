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

-- name: GetUserByEmail :one
select * from users where email = $1;

-- name: UpdateUser :one
with (select * from refresh_tokens where token = $1) as token_response
select * from users where id = token_response.user_id;
