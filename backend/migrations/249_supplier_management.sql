CREATE TABLE suppliers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    base_url VARCHAR(2048) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    notes TEXT NOT NULL DEFAULT '',
    balance_account_id BIGINT NULL REFERENCES accounts(id),
    balance_api_key TEXT NOT NULL DEFAULT '',
    balance NUMERIC(20,8) NULL,
    balance_status VARCHAR(20) NOT NULL DEFAULT 'unknown',
    balance_error TEXT NOT NULL DEFAULT '',
    balance_updated_at TIMESTAMPTZ NULL,
    balance_checked_at TIMESTAMPTZ NULL,
    revision BIGINT NOT NULL DEFAULT 1,
    balance_probe_sequence BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

ALTER TABLE accounts ADD COLUMN supplier_id BIGINT NULL REFERENCES suppliers(id);
ALTER TABLE accounts ADD COLUMN supplier_disabled BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX idx_accounts_supplier_id ON accounts(supplier_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_suppliers_status ON suppliers(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_suppliers_balance_account_id ON suppliers(balance_account_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_suppliers_deleted_at ON suppliers(deleted_at);
