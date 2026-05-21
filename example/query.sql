-- ── Queries handled by sqlc-gen-bulk-insert ────────────────────────────────

-- name: InsertUser :exec
-- Eligible: INSERT + :exec + multiple params → generates BulkInsertUser
INSERT INTO users (name, email, age, created_at) VALUES (?, ?, ?, ?);

-- name: InsertProduct :execrows
-- Eligible: INSERT + :execrows + multiple params → generates BulkInsertProduct
INSERT INTO products (sku, title, price_cents, in_stock, created_at) VALUES (?, ?, ?, ?, ?);

-- ── Queries NOT handled by sqlc-gen-bulk-insert ─────────────────────────────

-- name: GetUser :one
-- Skipped: not an INSERT (no InsertIntoTable set by sqlc)
SELECT id, name, email, age, created_at FROM users WHERE id = ? LIMIT 1;

-- name: ListUsers :many
-- Skipped: :many is not a no-output command
SELECT id, name, email FROM users ORDER BY id;
