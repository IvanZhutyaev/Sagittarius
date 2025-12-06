-- Create balances table
CREATE TABLE IF NOT EXISTS balances (
    user_id VARCHAR(255) PRIMARY KEY,
    balance BIGINT NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create reservations table
CREATE TABLE IF NOT EXISTS reservations (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    auction_id VARCHAR(255) NOT NULL,
    bid_id VARCHAR(255) NOT NULL,
    amount BIGINT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'reserved',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id),
    INDEX idx_auction_id (auction_id),
    INDEX idx_status (status)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_balances_user_id ON balances(user_id);
CREATE INDEX IF NOT EXISTS idx_reservations_user_id ON reservations(user_id);
CREATE INDEX IF NOT EXISTS idx_reservations_auction_id ON reservations(auction_id);
CREATE INDEX IF NOT EXISTS idx_reservations_status ON reservations(status);

-- Insert test user with balance
INSERT INTO balances (user_id, balance, version) 
VALUES ('test-user-1', 100000, 0)
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO balances (user_id, balance, version) 
VALUES ('test-user-2', 50000, 0)
ON CONFLICT (user_id) DO NOTHING;

