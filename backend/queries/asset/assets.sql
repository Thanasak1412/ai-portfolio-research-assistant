-- name: BootstrapCanonicalAsset :one
-- System-maintenance primitive only. It preserves an existing canonical ID and
-- refuses to silently change a canonical Asset's type or currency.
INSERT INTO assets (
    asset_id,
    symbol,
    name,
    asset_type,
    exchange,
    currency,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(asset_id),
    sqlc.arg(symbol),
    sqlc.arg(name),
    sqlc.arg(asset_type),
    sqlc.arg(exchange),
    'USD',
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
ON CONFLICT (normalized_symbol, normalized_exchange) DO UPDATE
SET
    symbol = EXCLUDED.symbol,
    name = EXCLUDED.name,
    exchange = EXCLUDED.exchange,
    updated_at = EXCLUDED.updated_at
WHERE
    assets.asset_type = EXCLUDED.asset_type
    AND assets.currency = EXCLUDED.currency
RETURNING *;

-- name: GetCanonicalAssetByID :one
SELECT *
FROM assets
WHERE asset_id = sqlc.arg(asset_id);

-- name: SearchCanonicalAssets :many
-- Cursor arguments are decoded, canonical ordering components. Cursor encoding
-- remains a transport/application responsibility.
SELECT *
FROM assets
WHERE
    (
        sqlc.narg(search)::text IS NULL
        OR symbol ILIKE '%' || sqlc.narg(search)::text || '%'
        OR name ILIKE '%' || sqlc.narg(search)::text || '%'
    )
    AND (
        sqlc.narg(asset_type)::text IS NULL
        OR asset_type = sqlc.narg(asset_type)::text
    )
    AND (
        sqlc.narg(cursor_symbol)::text IS NULL
        OR (symbol, exchange, asset_id) > (
            sqlc.narg(cursor_symbol)::text,
            sqlc.narg(cursor_exchange)::text,
            sqlc.narg(cursor_asset_id)::uuid
        )
    )
ORDER BY symbol ASC, exchange ASC, asset_id ASC
LIMIT sqlc.arg(page_limit);
