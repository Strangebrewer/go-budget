-- name: GetAllAccounts :many
SELECT * FROM accounts
WHERE user_id = $1
ORDER BY name;

-- name: GetAccountByID :one
SELECT * FROM accounts WHERE id = $1;

-- name: CreateAccount :one
INSERT INTO accounts (id, user_id, name, description, balance, owner, status, type, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateAccount :one
UPDATE accounts
SET name        = $2,
    description = $3,
    balance     = $4,
    owner       = $5,
    status      = $6,
    type        = $7,
    updated_at  = $8
WHERE id = $1
RETURNING *;

-- name: DeleteAccount :exec
DELETE FROM accounts WHERE id = $1;
