ALTER TABLE users
    ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'lender'
    CHECK (role IN ('lender', 'admin', 'investor'));

CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
