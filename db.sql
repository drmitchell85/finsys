CREATE TYPE transaction_status AS ENUM ('pending', 'processing', 'completed', 'failed', 'cancelled');
CREATE TYPE account_status AS ENUM ('active', 'suspended', 'closed', 'pending_verification');

CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    account_type VARCHAR(20) NOT NULL, -- merchant/customer/platform
    available_balance DECIMAL(15,2) DEFAULT 0, -- funds available for payout
    pending_balance DECIMAL(15,2) DEFAULT 0,   -- funds being processed
    currency VARCHAR(3) DEFAULT 'USD',
    status account_status NOT NULL DEFAULT 'active',
    external_bank_account_id VARCHAR(255), -- reference to their actual bank
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key VARCHAR(255) UNIQUE NOT NULL,
    from_account_id UUID NOT NULL,
    to_account_id UUID NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    status transaction_status NOT NULL DEFAULT 'pending',
    description TEXT,
    metadata JSONB,
    external_provider_id VARCHAR(255), -- stripe txn id, etc
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_transactions_from_account ON transactions(from_account_id);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_created_at ON transactions(created_at);