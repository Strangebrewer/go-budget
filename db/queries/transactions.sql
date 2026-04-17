-- name: GetAllTransactions :many
SELECT * FROM transactions
WHERE user_id = $1
ORDER BY date DESC, created_at DESC;

-- name: GetTransactionsByBillMonths :many
SELECT * FROM transactions
WHERE user_id = $1
  AND bill_month = ANY($2::text[])
ORDER BY bill_month, date;

-- name: GetTransactionsByCategories :many
SELECT * FROM transactions
WHERE user_id = $1
  AND category_id = ANY($2::uuid[])
ORDER BY date DESC, created_at DESC;

-- name: GetTransactionsByMonthAndCategories :many
SELECT * FROM transactions
WHERE user_id = $1
  AND date >= $2
  AND date < $3
  AND category_id = ANY($4::uuid[])
ORDER BY date DESC, created_at DESC;

-- name: GetIncomeTransactions :many
SELECT * FROM transactions
WHERE user_id = $1
  AND income = true
ORDER BY date DESC, created_at DESC;

-- name: GetIncomeTransactionsByMonth :many
SELECT * FROM transactions
WHERE user_id = $1
  AND income = true
  AND date >= $2
  AND date < $3
ORDER BY date DESC, created_at DESC;

-- name: GetTransactionByID :one
SELECT * FROM transactions WHERE id = $1;

-- name: CreateTransaction :one
INSERT INTO transactions (id, user_id, source_id, destination_id, bill_id, category_id, amount, bill_month, date, description, income, owner, shared, type, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
RETURNING *;

-- name: UpdateTransaction :one
UPDATE transactions
SET source_id      = $2,
    destination_id = $3,
    bill_id        = $4,
    category_id    = $5,
    amount         = $6,
    bill_month     = $7,
    date           = $8,
    description    = $9,
    income         = $10,
    owner          = $11,
    shared         = $12,
    type           = $13,
    updated_at     = $14
WHERE id = $1
RETURNING *;

-- name: DeleteTransaction :exec
DELETE FROM transactions WHERE id = $1;
