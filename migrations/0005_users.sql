-- Local operator accounts. This is a single-workstation desktop app, so users
-- exist to gate access to the local data, not to share it across machines.
-- Passwords are stored only as bcrypt hashes.
CREATE TABLE users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT    NOT NULL,
    password_hash TEXT    NOT NULL,
    display_name  TEXT    NOT NULL DEFAULT '',
    role          TEXT    NOT NULL DEFAULT 'admin' CHECK (role IN ('admin', 'staff')),
    created_at    TEXT    NOT NULL,
    updated_at    TEXT    NOT NULL,
    CHECK (length(trim(username)) > 0),
    CHECK (length(password_hash) > 0)
);

-- Usernames are unique case-insensitively so "Admin" cannot shadow "admin".
CREATE UNIQUE INDEX idx_users_username_unique ON users (username COLLATE NOCASE);
