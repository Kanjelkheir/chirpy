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
update users
set email = $2, password = $3, updated_at = $4
where id = (
    select user_id from refresh_tokens
    where token = $1 and revoked_at is null and expires_at > now()
)
returning *;

-- name: UpgradeUser :one
update users set is_chirpy_red = true where id = $1;
