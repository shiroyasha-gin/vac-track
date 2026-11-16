-- Accounts can be deactivated instead of deleted so audit history and foreign
-- references stay intact. Existing installations keep every current account
-- active.
ALTER TABLE users ADD COLUMN is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1));

CREATE INDEX idx_users_active ON users (is_active, username COLLATE NOCASE);
