-- TEST ONLY: synthetic catalog references, never provider market data.
-- Only scripts/seed-m3-e2e-assets.sh may load this into portfolio_test.
DO $$ BEGIN
    IF current_database() <> 'portfolio_test' THEN
        RAISE EXCEPTION 'M3 E2E fixtures require portfolio_test';
    END IF;
END $$;

INSERT INTO assets (
    asset_id, symbol, name, asset_type, exchange, currency, created_at, updated_at
) VALUES
    ('30000000-0000-4000-8000-000000000001', 'M3EQ01', 'Synthetic M3 Equity', 'EQUITY', 'NYSE', 'USD', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
    ('30000000-0000-4000-8000-000000000002', 'M3ETF01', 'Synthetic M3 ETF', 'ETF', 'NYSEARCA', 'USD', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
    ('30000000-0000-4000-8000-000000000003', 'M3CR01', 'Synthetic M3 Crypto', 'CRYPTO', 'CRYPTO', 'USD', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')
ON CONFLICT (normalized_symbol, normalized_exchange) DO NOTHING;
