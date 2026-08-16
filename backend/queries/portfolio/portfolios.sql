-- name: CreatePortfolio :one
INSERT INTO portfolios (
    portfolio_id,
    owner_user_id,
    name,
    base_currency,
    status,
    archived_at,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(portfolio_id),
    sqlc.arg(owner_user_id),
    sqlc.arg(name),
    'USD',
    'ACTIVE',
    NULL,
    sqlc.arg(created_at),
    sqlc.arg(created_at)
)
RETURNING *;

-- name: GetOwnedPortfolioByID :one
SELECT *
FROM portfolios
WHERE
    portfolio_id = sqlc.arg(portfolio_id)
    AND owner_user_id = sqlc.arg(owner_user_id);

-- name: ListOwnedPortfoliosByStatus :many
SELECT *
FROM portfolios
WHERE
    owner_user_id = sqlc.arg(owner_user_id)
    AND status = sqlc.arg(status)
ORDER BY updated_at DESC, portfolio_id ASC;

-- name: UpdateOwnedActivePortfolioName :one
-- The caller must classify no-row and uniqueness outcomes without exposing
-- database details as public HTTP errors.
UPDATE portfolios
SET
    name = sqlc.arg(name),
    updated_at = sqlc.arg(updated_at)
WHERE
    portfolio_id = sqlc.arg(portfolio_id)
    AND owner_user_id = sqlc.arg(owner_user_id)
    AND status = 'ACTIVE'
RETURNING *;

-- name: ArchiveOwnedActivePortfolio :one
-- An archive retry is resolved by the future application layer through the
-- owner-scoped read primitive; this mutation intentionally changes ACTIVE only.
UPDATE portfolios
SET
    status = 'ARCHIVED',
    archived_at = sqlc.arg(archived_at),
    updated_at = sqlc.arg(updated_at)
WHERE
    portfolio_id = sqlc.arg(portfolio_id)
    AND owner_user_id = sqlc.arg(owner_user_id)
    AND status = 'ACTIVE'
RETURNING *;
