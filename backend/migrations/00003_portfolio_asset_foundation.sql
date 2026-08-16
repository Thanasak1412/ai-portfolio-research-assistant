-- +goose Up
-- M2 authoritative Portfolio and canonical Asset persistence. IDs are generated
-- by trusted application code, consistently with the existing Identity tables.

CREATE TABLE portfolios (
    portfolio_id uuid PRIMARY KEY,
    owner_user_id uuid NOT NULL,
    name text NOT NULL,
    normalized_name text GENERATED ALWAYS AS (
        lower(btrim(name, E' \t\n\r\f\013'))
    ) STORED,
    base_currency text NOT NULL,
    status text NOT NULL,
    archived_at timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT portfolios_owner_user_fk
        FOREIGN KEY (owner_user_id) REFERENCES users (user_id) ON DELETE RESTRICT,
    CONSTRAINT portfolios_name_trimmed CHECK (
        name = btrim(name, E' \t\n\r\f\013')
        AND name <> ''
        AND char_length(name) <= 200
    ),
    CONSTRAINT portfolios_base_currency_usd CHECK (base_currency = 'USD'),
    CONSTRAINT portfolios_status_valid CHECK (status IN ('ACTIVE', 'ARCHIVED')),
    CONSTRAINT portfolios_archive_state_consistent CHECK (
        (status = 'ACTIVE' AND archived_at IS NULL)
        OR (status = 'ARCHIVED' AND archived_at IS NOT NULL)
    ),
    CONSTRAINT portfolios_timestamps_ordered CHECK (updated_at >= created_at),
    CONSTRAINT portfolios_archived_timestamp_ordered CHECK (
        archived_at IS NULL OR archived_at >= created_at
    )
);

-- This is the final concurrency authority for active-name uniqueness. Archived
-- records deliberately do not participate, so archiving releases the name.
CREATE UNIQUE INDEX portfolios_owner_normalized_active_uidx
    ON portfolios (owner_user_id, normalized_name)
    WHERE status = 'ACTIVE';

-- Serves owner-scoped ACTIVE/ARCHIVED listing in the frozen API order.
CREATE INDEX portfolios_owner_status_updated_id_idx
    ON portfolios (owner_user_id, status, updated_at DESC, portfolio_id ASC);

CREATE TABLE assets (
    asset_id uuid PRIMARY KEY,
    symbol text NOT NULL,
    normalized_symbol text GENERATED ALWAYS AS (
        upper(btrim(symbol, E' \t\n\r\f\013'))
    ) STORED,
    name text NOT NULL,
    asset_type text NOT NULL,
    exchange text NOT NULL,
    normalized_exchange text GENERATED ALWAYS AS (
        upper(btrim(exchange, E' \t\n\r\f\013'))
    ) STORED,
    currency text NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT assets_symbol_trimmed CHECK (
        symbol = btrim(symbol, E' \t\n\r\f\013')
        AND symbol <> ''
        AND char_length(symbol) <= 64
    ),
    CONSTRAINT assets_name_not_empty CHECK (
        btrim(name, E' \t\n\r\f\013') <> ''
        AND char_length(name) <= 256
    ),
    CONSTRAINT assets_exchange_trimmed CHECK (
        exchange = btrim(exchange, E' \t\n\r\f\013')
        AND exchange <> ''
        AND char_length(exchange) <= 64
    ),
    CONSTRAINT assets_type_valid CHECK (asset_type IN ('EQUITY', 'ETF', 'CRYPTO')),
    CONSTRAINT assets_currency_usd CHECK (currency = 'USD'),
    CONSTRAINT assets_crypto_exchange_consistent CHECK (
        asset_type <> 'CRYPTO'
        OR (exchange = 'CRYPTO' AND normalized_exchange = 'CRYPTO')
    ),
    CONSTRAINT assets_timestamps_ordered CHECK (updated_at >= created_at),
    CONSTRAINT assets_normalized_identity_unique UNIQUE (
        normalized_symbol,
        normalized_exchange
    )
);

-- Serves the frozen public Asset order and keyset continuation:
-- symbol, exchange, then id ascending. The canonical identity unique index
-- separately serves normalized symbol/exchange conflict detection.
CREATE INDEX assets_symbol_exchange_id_idx
    ON assets (symbol ASC, exchange ASC, asset_id ASC);

-- +goose Down
DROP TABLE assets;
DROP TABLE portfolios;
