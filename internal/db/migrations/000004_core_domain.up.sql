CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS lenders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    country_code CHAR(2) NOT NULL,
    wallet_address CHAR(42) UNIQUE NOT NULL,
    kyc_status TEXT NOT NULL DEFAULT 'pending' CHECK (kyc_status IN ('pending','approved','suspended')),
    tier TEXT NOT NULL DEFAULT 'starter' CHECK (tier IN ('starter','growth','enterprise')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS borrowers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    borrower_hash BYTEA UNIQUE NOT NULL,
    lender_id UUID REFERENCES lenders(id),
    country_code CHAR(2) NOT NULL,
    sector TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_borrowers_hash ON borrowers(borrower_hash);

CREATE TABLE IF NOT EXISTS loans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    loan_hash BYTEA UNIQUE NOT NULL,
    lender_id UUID NOT NULL REFERENCES lenders(id),
    borrower_id UUID NOT NULL REFERENCES borrowers(id),
    principal_minor BIGINT NOT NULL,
    currency_code CHAR(3) NOT NULL,
    interest_rate_bps INT NOT NULL,
    start_date TIMESTAMPTZ NOT NULL,
    maturity_date TIMESTAMPTZ NOT NULL,
    amount_repaid_minor BIGINT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','repaid','defaulted')),
    on_chain_tx CHAR(66),
    on_chain_confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    risk_grade CHAR(1) CHECK (risk_grade IN ('A','B','C')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_loans_lender ON loans(lender_id);
CREATE INDEX IF NOT EXISTS idx_loans_borrower ON loans(borrower_id);
CREATE INDEX IF NOT EXISTS idx_loans_status ON loans(status);
CREATE INDEX IF NOT EXISTS idx_loans_maturity ON loans(maturity_date);

CREATE TABLE IF NOT EXISTS repayments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    loan_id UUID NOT NULL REFERENCES loans(id),
    amount_minor BIGINT NOT NULL,
    on_chain_tx CHAR(66),
    on_chain_event JSONB,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_repayments_loan ON repayments(loan_id);

CREATE TABLE IF NOT EXISTS passport_cache (
    borrower_id UUID PRIMARY KEY REFERENCES borrowers(id),
    token_id BIGINT,
    total_loans INT NOT NULL DEFAULT 0,
    total_repaid INT NOT NULL DEFAULT 0,
    total_defaulted INT NOT NULL DEFAULT 0,
    cumulative_borrowed BIGINT NOT NULL DEFAULT 0,
    cumulative_repaid BIGINT NOT NULL DEFAULT 0,
    credit_score INT NOT NULL DEFAULT 300 CHECK (credit_score BETWEEN 300 AND 850),
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lender_id UUID NOT NULL REFERENCES lenders(id),
    name TEXT NOT NULL,
    pool_token_addr CHAR(42),
    target_apybps INT NOT NULL,
    currency_code CHAR(3) NOT NULL,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','closed','settled')),
    total_deployed BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_pools_lender ON pools(lender_id);
CREATE INDEX IF NOT EXISTS idx_pools_status ON pools(status);

CREATE TABLE IF NOT EXISTS chain_events (
    id BIGSERIAL PRIMARY KEY,
    contract_addr CHAR(42) NOT NULL,
    event_name TEXT NOT NULL,
    tx_hash CHAR(66) NOT NULL,
    block_number BIGINT NOT NULL,
    log_index INT NOT NULL,
    raw_data JSONB NOT NULL,
    processed BOOLEAN NOT NULL DEFAULT FALSE,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tx_hash, log_index)
);
CREATE INDEX IF NOT EXISTS idx_events_processed ON chain_events(processed);
CREATE INDEX IF NOT EXISTS idx_events_event ON chain_events(event_name);

CREATE TABLE IF NOT EXISTS outbox_jobs (
    id BIGSERIAL PRIMARY KEY,
    topic TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','processing','done','failed')),
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_outbox_jobs_status_available_at ON outbox_jobs(status, available_at);
