-- +goose Up
-- Transaction-owned authoritative facts and operational command identity.
-- No triggers, projections, cross-module writes, or financial replay in SQL.
CREATE TABLE transaction_portfolio_sequences (
    portfolio_id uuid PRIMARY KEY REFERENCES portfolios (portfolio_id) ON DELETE RESTRICT,
    next_sequence bigint NOT NULL CHECK (next_sequence > 0)
);

CREATE TABLE transactions (
    transaction_id uuid PRIMARY KEY,
    portfolio_id uuid NOT NULL REFERENCES portfolios (portfolio_id) ON DELETE RESTRICT,
    created_by_user_id uuid NOT NULL REFERENCES users (user_id) ON DELETE RESTRICT,
    kind text NOT NULL CHECK (kind IN ('BUY', 'SELL', 'DIVIDEND', 'DEPOSIT', 'WITHDRAWAL', 'FEE', 'REVERSAL')),
    asset_id uuid REFERENCES assets (asset_id) ON DELETE RESTRICT,
    asset_type_snapshot text,
    asset_exchange_snapshot text,
    asset_currency_snapshot text,
    -- Unconstrained NUMERIC avoids rounding at assignment or an invented
    -- significant-digit bound. CHECKs reject nonfinite values and excess scale.
    quantity numeric CHECK (quantity > 0 AND quantity < 'Infinity'::numeric AND scale(quantity) <= 12),
    unit_price numeric CHECK (unit_price > 0 AND unit_price < 'Infinity'::numeric AND scale(unit_price) <= 12),
    fee numeric CHECK (fee >= 0 AND fee < 'Infinity'::numeric AND scale(fee) <= 12),
    amount numeric CHECK (amount > 0 AND amount < 'Infinity'::numeric AND scale(amount) <= 12),
    currency text NOT NULL CHECK (currency = 'USD'),
    effective_at timestamptz NOT NULL CHECK (isfinite(effective_at)),
    portfolio_sequence bigint NOT NULL CHECK (portfolio_sequence > 0),
    note text CHECK (char_length(note) <= 2000),
    external_reference text CHECK (char_length(external_reference) <= 256),
    reversal_of_transaction_id uuid,
    correction_of_transaction_id uuid,
    originating_correction_id uuid,
    created_at timestamptz NOT NULL CHECK (isfinite(created_at)),
    CONSTRAINT transactions_portfolio_sequence_unique UNIQUE (portfolio_id, portfolio_sequence),
    -- Composite reference keys preserve scope and enforce complete corrections.
    CONSTRAINT transactions_scoped_identity UNIQUE (portfolio_id, transaction_id),
    CONSTRAINT transactions_reversal_identity UNIQUE (portfolio_id, transaction_id, reversal_of_transaction_id, originating_correction_id),
    CONSTRAINT transactions_replacement_identity UNIQUE (portfolio_id, transaction_id, correction_of_transaction_id, originating_correction_id),
    CONSTRAINT transactions_asset_snapshot CHECK (
        (asset_id IS NULL AND asset_type_snapshot IS NULL AND asset_exchange_snapshot IS NULL AND asset_currency_snapshot IS NULL)
        OR (asset_id IS NOT NULL AND asset_type_snapshot IS NOT NULL AND asset_exchange_snapshot IS NOT NULL AND asset_currency_snapshot IS NOT NULL
            AND asset_type_snapshot IN ('EQUITY', 'ETF')
            AND asset_exchange_snapshot IN ('NYSE', 'NASDAQ', 'NYSEARCA', 'AMEX') AND asset_currency_snapshot = 'USD')
    ),
    CONSTRAINT transactions_field_matrix CHECK (
        (kind IN ('BUY', 'SELL', 'REVERSAL') AND asset_id IS NOT NULL AND quantity IS NOT NULL AND unit_price IS NOT NULL AND fee IS NOT NULL AND amount IS NULL)
        OR (kind IN ('DIVIDEND', 'REVERSAL') AND asset_id IS NOT NULL AND amount IS NOT NULL AND quantity IS NULL AND unit_price IS NULL AND fee IS NULL)
        OR (kind IN ('DEPOSIT', 'WITHDRAWAL', 'FEE', 'REVERSAL') AND asset_id IS NULL AND amount IS NOT NULL AND quantity IS NULL AND unit_price IS NULL AND fee IS NULL)
    ),
    CONSTRAINT transactions_creation_links CHECK (
        (kind = 'REVERSAL' AND reversal_of_transaction_id IS NOT NULL AND correction_of_transaction_id IS NULL AND originating_correction_id IS NOT NULL AND note IS NULL AND external_reference IS NULL)
        OR (kind <> 'REVERSAL' AND reversal_of_transaction_id IS NULL
            AND ((correction_of_transaction_id IS NULL AND originating_correction_id IS NULL)
                OR (correction_of_transaction_id IS NOT NULL AND originating_correction_id IS NOT NULL)))
    ),
    CONSTRAINT transactions_not_self_reversal CHECK (transaction_id <> reversal_of_transaction_id),
    CONSTRAINT transactions_not_self_replacement CHECK (transaction_id <> correction_of_transaction_id),
    CONSTRAINT transactions_reversal_original_fk FOREIGN KEY (portfolio_id, reversal_of_transaction_id)
        REFERENCES transactions (portfolio_id, transaction_id) ON DELETE RESTRICT,
    CONSTRAINT transactions_replacement_original_fk FOREIGN KEY (portfolio_id, correction_of_transaction_id)
        REFERENCES transactions (portfolio_id, transaction_id) ON DELETE RESTRICT
);

-- History/keyset and transient asset-ledger replay, respectively.
CREATE INDEX transactions_history_idx ON transactions (portfolio_id, effective_at DESC, portfolio_sequence DESC, transaction_id DESC);
CREATE INDEX transactions_asset_replay_idx ON transactions (portfolio_id, asset_id, effective_at ASC, portfolio_sequence ASC, transaction_id ASC);
CREATE UNIQUE INDEX transactions_direct_reversal_uidx ON transactions (portfolio_id, reversal_of_transaction_id) WHERE reversal_of_transaction_id IS NOT NULL;
CREATE UNIQUE INDEX transactions_direct_replacement_uidx ON transactions (portfolio_id, correction_of_transaction_id) WHERE correction_of_transaction_id IS NOT NULL;

CREATE TABLE transaction_corrections (
    correction_id uuid PRIMARY KEY,
    portfolio_id uuid NOT NULL REFERENCES portfolios (portfolio_id) ON DELETE RESTRICT,
    original_transaction_id uuid NOT NULL,
    reversal_transaction_id uuid NOT NULL,
    replacement_transaction_id uuid NOT NULL,
    created_by_user_id uuid NOT NULL REFERENCES users (user_id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL CHECK (isfinite(created_at)),
    CONSTRAINT corrections_distinct CHECK (original_transaction_id <> reversal_transaction_id AND original_transaction_id <> replacement_transaction_id AND reversal_transaction_id <> replacement_transaction_id),
    CONSTRAINT corrections_one_original UNIQUE (portfolio_id, original_transaction_id),
    CONSTRAINT corrections_one_reversal UNIQUE (portfolio_id, reversal_transaction_id),
    CONSTRAINT corrections_one_replacement UNIQUE (portfolio_id, replacement_transaction_id),
    CONSTRAINT corrections_reversal_identity UNIQUE (portfolio_id, correction_id, original_transaction_id, reversal_transaction_id),
    CONSTRAINT corrections_replacement_identity UNIQUE (portfolio_id, correction_id, original_transaction_id, replacement_transaction_id),
    CONSTRAINT corrections_original_fk FOREIGN KEY (portfolio_id, original_transaction_id)
        REFERENCES transactions (portfolio_id, transaction_id) ON DELETE RESTRICT,
    CONSTRAINT corrections_reversal_fk FOREIGN KEY (portfolio_id, reversal_transaction_id, original_transaction_id, correction_id)
        REFERENCES transactions (portfolio_id, transaction_id, reversal_of_transaction_id, originating_correction_id) ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT corrections_replacement_fk FOREIGN KEY (portfolio_id, replacement_transaction_id, original_transaction_id, correction_id)
        REFERENCES transactions (portfolio_id, transaction_id, correction_of_transaction_id, originating_correction_id) ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED
);

-- Both directions are necessary: neither a correction missing a fact nor an
-- orphan reversal/replacement can commit. Original records are never updated.
ALTER TABLE transactions ADD CONSTRAINT transactions_complete_reversal_fk
    FOREIGN KEY (portfolio_id, originating_correction_id, reversal_of_transaction_id, transaction_id)
    REFERENCES transaction_corrections (portfolio_id, correction_id, original_transaction_id, reversal_transaction_id)
    ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED;
ALTER TABLE transactions ADD CONSTRAINT transactions_complete_replacement_fk
    FOREIGN KEY (portfolio_id, originating_correction_id, correction_of_transaction_id, transaction_id)
    REFERENCES transaction_corrections (portfolio_id, correction_id, original_transaction_id, replacement_transaction_id)
    ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE transaction_idempotency (
    portfolio_id uuid NOT NULL REFERENCES portfolios (portfolio_id) ON DELETE RESTRICT,
    command_scope text NOT NULL CHECK (command_scope IN ('transaction.create.v1', 'transaction.correct.v1')),
    idempotency_key text NOT NULL CHECK (idempotency_key ~ '^[A-Za-z0-9][A-Za-z0-9._~-]{15,127}$'),
    fingerprint_digest bytea NOT NULL CHECK (octet_length(fingerprint_digest) = 32),
    target_transaction_id uuid,
    primary_transaction_id uuid NOT NULL,
    correction_id uuid,
    created_at timestamptz NOT NULL CHECK (isfinite(created_at)),
    expires_at timestamptz NOT NULL CHECK (isfinite(expires_at) AND expires_at > created_at),
    PRIMARY KEY (portfolio_id, command_scope, idempotency_key),
    CONSTRAINT idempotency_completed_shape CHECK (
        (command_scope = 'transaction.create.v1' AND target_transaction_id IS NULL AND correction_id IS NULL)
        OR (command_scope = 'transaction.correct.v1' AND target_transaction_id IS NOT NULL AND correction_id IS NOT NULL)
    ),
    CONSTRAINT idempotency_result_fk FOREIGN KEY (portfolio_id, primary_transaction_id)
        REFERENCES transactions (portfolio_id, transaction_id) ON DELETE RESTRICT,
    CONSTRAINT idempotency_correction_result_fk FOREIGN KEY (portfolio_id, correction_id, target_transaction_id, primary_transaction_id)
        REFERENCES transaction_corrections (portfolio_id, correction_id, original_transaction_id, replacement_transaction_id) ON DELETE RESTRICT
);
CREATE INDEX transaction_idempotency_expiry_idx ON transaction_idempotency (expires_at, portfolio_id, command_scope, idempotency_key);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    -- Hold locks through Goose's migration transaction to close the preflight
    -- race with a concurrent append. Rollback never silently deletes evidence.
    LOCK TABLE transaction_portfolio_sequences, transactions, transaction_corrections, transaction_idempotency IN ACCESS EXCLUSIVE MODE;
    IF EXISTS (SELECT 1 FROM transactions) OR EXISTS (SELECT 1 FROM transaction_corrections)
        OR EXISTS (SELECT 1 FROM transaction_idempotency) OR EXISTS (SELECT 1 FROM transaction_portfolio_sequences) THEN
        RAISE EXCEPTION 'M3 ledger rollback refused: retained Transaction state requires an approved retention/archive decision';
    END IF;
END $$;
-- +goose StatementEnd
DROP TABLE transaction_idempotency;
ALTER TABLE transactions DROP CONSTRAINT transactions_complete_reversal_fk;
ALTER TABLE transactions DROP CONSTRAINT transactions_complete_replacement_fk;
DROP TABLE transaction_corrections;
DROP TABLE transactions;
DROP TABLE transaction_portfolio_sequences;
