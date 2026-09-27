-- name: InsertTransactionCorrection :one
-- Caller-owned pgx.Tx; deferred FKs require matching reversal and replacement
-- rows before commit. No original financial record is mutated.
INSERT INTO transaction_corrections (
    correction_id, portfolio_id, original_transaction_id, reversal_transaction_id,
    replacement_transaction_id, created_by_user_id, created_at
) VALUES (
    sqlc.arg(correction_id), sqlc.arg(portfolio_id), sqlc.arg(original_transaction_id), sqlc.arg(reversal_transaction_id),
    sqlc.arg(replacement_transaction_id), sqlc.arg(created_by_user_id), sqlc.arg(created_at)
) RETURNING *;

-- name: GetPortfolioCorrection :one
SELECT * FROM transaction_corrections
WHERE portfolio_id = sqlc.arg(portfolio_id) AND correction_id = sqlc.arg(correction_id);

-- name: GetDirectTransactionCorrection :one
-- Supplies outgoing links without changing the original. A replacement retains
-- its incoming creation links and can independently have an outgoing correction.
SELECT * FROM transaction_corrections
WHERE portfolio_id = sqlc.arg(portfolio_id) AND original_transaction_id = sqlc.arg(original_transaction_id);
