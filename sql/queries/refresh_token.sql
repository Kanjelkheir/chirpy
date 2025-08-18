-- name: CreateRefreshToken :one
insert into refresh_tokens (token, created_at, updated_at, user_id, expires_at, revoked_at) values(
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
) returning *;


-- name: GetRefreshToken :one
select * from refresh_tokens where token = $1 and revoked_at is null and expires_at > now();


-- name: RevokeRefreshToken :exec
update refresh_tokens
set revoked_at = $1, updated_at = $2
where token = $3;
