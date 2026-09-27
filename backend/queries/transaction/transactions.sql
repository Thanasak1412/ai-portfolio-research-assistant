-- name: InsertTransaction :one
-- Caller-owned transaction after idempotency + sequence locks and application
-- validation. Missing trade fee is supplied as exact zero by that application.
-- Reversal/replacement rows must commit with their complete correction group.
INSERT INTO transactions (
    transaction_id, portfolio_id, created_by_user_id, kind, asset_id,
    asset_type_snapshot, asset_exchange_snapshot, asset_currency_snapshot,
    quantity, unit_price, fee, amount, currency, effective_at, portfolio_sequence,
    note, external_reference, reversal_of_transaction_id,
    correction_of_transaction_id, originating_correction_id, created_at
) VALUES (
    sqlc.arg(transaction_id), sqlc.arg(portfolio_id), sqlc.arg(created_by_user_id), sqlc.arg(kind), sqlc.narg(asset_id),
    sqlc.narg(asset_type_snapshot), sqlc.narg(asset_exchange_snapshot), sqlc.narg(asset_currency_snapshot),
    sqlc.narg(quantity), sqlc.narg(unit_price), sqlc.narg(fee), sqlc.narg(amount), sqlc.arg(currency), sqlc.arg(effective_at), sqlc.arg(portfolio_sequence),
    sqlc.narg(note), sqlc.narg(external_reference), sqlc.narg(reversal_of_transaction_id),
    sqlc.narg(correction_of_transaction_id), sqlc.narg(originating_correction_id), sqlc.arg(created_at)
) RETURNING *;

-- name: GetPortfolioTransaction :one
-- Ownership must be proven through the Portfolio public boundary BEFORE use.
SELECT * FROM transactions
WHERE portfolio_id = sqlc.arg(portfolio_id) AND transaction_id = sqlc.arg(transaction_id);

-- name: ListPortfolioTransactions :many
-- Caller validates filters/opaque cursor and supplies the full decoded tuple
-- or three NULLs. Page limit is the contract limit (1..100); no offset paging.
SELECT * FROM transactions
WHERE portfolio_id = sqlc.arg(portfolio_id)
    AND (sqlc.narg(kind)::text IS NULL OR kind = sqlc.narg(kind))
    AND (sqlc.narg(effective_at_from)::timestamptz IS NULL OR effective_at >= sqlc.narg(effective_at_from))
    AND (sqlc.narg(effective_at_to)::timestamptz IS NULL OR effective_at <= sqlc.narg(effective_at_to))
    AND (sqlc.arg(include_reversals)::boolean OR kind <> 'REVERSAL')
    AND (sqlc.narg(cursor_effective_at)::timestamptz IS NULL OR
        (effective_at, portfolio_sequence, transaction_id) <
        (sqlc.narg(cursor_effective_at)::timestamptz, sqlc.narg(cursor_portfolio_sequence)::bigint, sqlc.narg(cursor_transaction_id)::uuid))
ORDER BY effective_at DESC, portfolio_sequence DESC, transaction_id DESC
LIMIT sqlc.arg(page_limit)::int;

-- name: ListAssetLedgerForReplay :many
-- Caller MUST hold the Portfolio sequence lock in the same pgx.Tx before
-- reading. Raw immutable facts only: sign/equality/quantity replay stays Go-side.
SELECT * FROM transactions
WHERE portfolio_id = sqlc.arg(portfolio_id) AND asset_id = sqlc.arg(asset_id)
ORDER BY effective_at ASC, portfolio_sequence ASC, transaction_id ASC;
