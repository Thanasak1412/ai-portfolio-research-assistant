-- name: AllocateOnePortfolioSequence :one
-- Caller MUST hold the idempotency advisory lock first and use a caller-owned
-- pgx.Tx. This UPSERT locks the Portfolio stream until commit/rollback; read
-- the replay facts only AFTER allocation. Never allocate in autocommit mode.
INSERT INTO transaction_portfolio_sequences (portfolio_id, next_sequence)
VALUES (sqlc.arg(portfolio_id), 2)
ON CONFLICT (portfolio_id) DO UPDATE
SET next_sequence = transaction_portfolio_sequences.next_sequence + 1
RETURNING (next_sequence - 1)::bigint AS portfolio_sequence;

-- name: AllocateCorrectionPortfolioSequences :one
-- Same lock order/transaction requirement as AllocateOnePortfolioSequence.
-- Reserve reversal then replacement; rollback restores the counter.
INSERT INTO transaction_portfolio_sequences (portfolio_id, next_sequence)
VALUES (sqlc.arg(portfolio_id), 3)
ON CONFLICT (portfolio_id) DO UPDATE
SET next_sequence = transaction_portfolio_sequences.next_sequence + 2
RETURNING (next_sequence - 2)::bigint AS reversal_sequence,
    (next_sequence - 1)::bigint AS replacement_sequence;
