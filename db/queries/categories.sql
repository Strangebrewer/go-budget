-- name: GetAllCategories :many
SELECT * FROM categories
WHERE user_id = $1
ORDER BY name;

-- name: GetCategoryByID :one
SELECT * FROM categories WHERE id = $1;

-- name: CreateCategory :one
INSERT INTO categories (id, user_id, name, description, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateCategory :one
UPDATE categories
SET name        = $2,
    description = $3,
    updated_at  = $4
WHERE id = $1
RETURNING *;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE id = $1;
