-- name: GetAllBills :many
SELECT * FROM bills
WHERE user_id = $1 AND status = 'active'
ORDER BY name;

-- name: GetBillByID :one
SELECT * FROM bills WHERE id = $1;

-- name: CreateBill :one
INSERT INTO bills (id, user_id, source_id, category_id, name, description, due_day, owner, shared, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: UpdateBill :one
UPDATE bills
SET source_id   = $2,
    category_id = $3,
    name        = $4,
    description = $5,
    due_day     = $6,
    owner       = $7,
    shared      = $8,
    status      = $9,
    updated_at  = $10
WHERE id = $1
RETURNING *;

-- name: DeleteBill :exec
DELETE FROM bills WHERE id = $1;
