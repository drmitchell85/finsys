CREATE TYPE user_status AS ENUM ('active', 'suspended', 'deleted');
CREATE TYPE transaction_status AS ENUM ('pending', 'processing', 'completed', 'failed', 'cancelled');
CREATE TYPE account_status AS ENUM ('active', 'suspended', 'closed', 'pending_verification');
CREATE TYPE mock_account_status AS ENUM ('active', 'suspended', 'closed', 'pending_verification');
CREATE TYPE account_type AS ENUM ('merchant', 'customer', 'platform');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    status user_status NOT NULL DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

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

CREATE TABLE mock_accounts (
   id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
   balance DECIMAL(15,2) NOT NULL DEFAULT 0,
   currency VARCHAR(3) NOT NULL DEFAULT 'USD',
   status mock_account_status NOT NULL DEFAULT 'active',
   updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE mock_reservations (
   id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
   account_id UUID NOT NULL REFERENCES mock_accounts(id),
   amount DECIMAL(15,2) NOT NULL,
   expires_at TIMESTAMP NOT NULL,
   created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_transactions_from_account ON transactions(from_account_id);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_created_at ON transactions(created_at);

ALTER TABLE transactions ADD COLUMN bank_reservation_id UUID;

ALTER TABLE accounts ADD CONSTRAINT fk_accounts_user_id 
FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE accounts ALTER COLUMN account_type TYPE account_type USING account_type::account_type;

ALTER TABLE accounts ALTER COLUMN external_bank_account_id TYPE UUID USING external_bank_account_id::UUID;

ALTER TABLE accounts ADD CONSTRAINT fk_accounts_external_bank 
FOREIGN KEY (external_bank_account_id) REFERENCES mock_accounts(id);

ALTER TABLE transactions ADD CONSTRAINT fk_transactions_reservation 
FOREIGN KEY (bank_reservation_id) REFERENCES mock_reservations(id);

ALTER TABLE accounts DROP CONSTRAINT fk_accounts_external_bank;
ALTER TABLE transactions DROP CONSTRAINT fk_transactions_reservation;