CREATE TABLE IF NOT EXISTS exchange_snapshots (
    id BIGSERIAL PRIMARY KEY,
    provider TEXT NOT NULL,
    account_scope TEXT NOT NULL DEFAULT 'spot',
    payload JSONB NOT NULL,
    collected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exchange_snapshots_provider_collected_at
    ON exchange_snapshots (provider, collected_at DESC);

