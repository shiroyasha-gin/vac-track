-- Vaccination inventory tracker: initial schema.
-- All quantities are whole doses (INTEGER). Dates are stored as:
--   timestamp: RFC3339 UTC text
--   calendar date: YYYY-MM-DD text
--   inventory month: YYYY-MM text
-- Foreign keys are enabled per connection by the application.

CREATE TABLE vaccines (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL,
    code        TEXT    NOT NULL DEFAULT '',
    description TEXT    NOT NULL DEFAULT '',
    category    TEXT    NOT NULL DEFAULT '',
    unit        TEXT    NOT NULL DEFAULT 'dose',
    is_archived INTEGER NOT NULL DEFAULT 0 CHECK (is_archived IN (0, 1)),
    archived_at TEXT,
    created_at  TEXT    NOT NULL,
    updated_at  TEXT    NOT NULL,
    CHECK (length(trim(name)) > 0),
    CHECK (length(trim(unit)) > 0)
);

CREATE UNIQUE INDEX idx_vaccines_name_unique ON vaccines (name COLLATE NOCASE);
CREATE UNIQUE INDEX idx_vaccines_code_unique ON vaccines (code COLLATE NOCASE) WHERE code <> '';
CREATE INDEX idx_vaccines_archived ON vaccines (is_archived, name COLLATE NOCASE);

CREATE TABLE patients (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    full_name    TEXT    NOT NULL,
    patient_code TEXT    NOT NULL DEFAULT '',
    is_archived  INTEGER NOT NULL DEFAULT 0 CHECK (is_archived IN (0, 1)),
    archived_at  TEXT,
    created_at   TEXT    NOT NULL,
    updated_at   TEXT    NOT NULL,
    CHECK (length(trim(full_name)) > 0)
);

CREATE UNIQUE INDEX idx_patients_code_unique ON patients (patient_code COLLATE NOCASE) WHERE patient_code <> '';
CREATE INDEX idx_patients_name ON patients (full_name COLLATE NOCASE);
CREATE INDEX idx_patients_archived ON patients (is_archived, full_name COLLATE NOCASE);

CREATE TABLE monthly_inventory (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    vaccine_id            INTEGER NOT NULL REFERENCES vaccines (id),
    month                 TEXT    NOT NULL CHECK (
        length(month) = 7 AND month GLOB '[0-9][0-9][0-9][0-9]-[0-1][0-9]'
    ),
    opening_balance       INTEGER NOT NULL CHECK (opening_balance >= 0),
    stock_received        INTEGER NOT NULL CHECK (stock_received >= 0),
    status                TEXT    NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    closing_balance       INTEGER CHECK (closing_balance IS NULL OR closing_balance >= 0),
    carryover_overridden  INTEGER NOT NULL DEFAULT 0 CHECK (carryover_overridden IN (0, 1)),
    carryover_reason      TEXT    NOT NULL DEFAULT '',
    closed_at             TEXT,
    reopened_at           TEXT,
    created_at            TEXT    NOT NULL,
    updated_at            TEXT    NOT NULL,
    UNIQUE (vaccine_id, month)
);

CREATE INDEX idx_monthly_inventory_month ON monthly_inventory (month, vaccine_id);
CREATE INDEX idx_monthly_inventory_status ON monthly_inventory (month, status);

CREATE TABLE usage_entries (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    patient_id       INTEGER NOT NULL REFERENCES patients (id),
    vaccine_id       INTEGER NOT NULL REFERENCES vaccines (id),
    administered_on  TEXT    NOT NULL CHECK (
        length(administered_on) = 10 AND administered_on GLOB '[0-9][0-9][0-9][0-9]-[0-1][0-9]-[0-3][0-9]'
    ),
    month            TEXT    NOT NULL CHECK (
        length(month) = 7 AND month GLOB '[0-9][0-9][0-9][0-9]-[0-1][0-9]'
    ),
    quantity         INTEGER NOT NULL CHECK (quantity > 0),
    note             TEXT    NOT NULL DEFAULT '',
    voided_at        TEXT,
    void_reason      TEXT    NOT NULL DEFAULT '',
    created_at       TEXT    NOT NULL,
    updated_at       TEXT    NOT NULL
);

CREATE INDEX idx_usage_vaccine_month ON usage_entries (vaccine_id, month, voided_at);
CREATE INDEX idx_usage_vaccine_date ON usage_entries (vaccine_id, administered_on);
CREATE INDEX idx_usage_patient_date ON usage_entries (patient_id, administered_on);
CREATE INDEX idx_usage_date ON usage_entries (administered_on);

CREATE TABLE inventory_adjustments (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    vaccine_id     INTEGER NOT NULL REFERENCES vaccines (id),
    month          TEXT    NOT NULL CHECK (
        length(month) = 7 AND month GLOB '[0-9][0-9][0-9][0-9]-[0-1][0-9]'
    ),
    amount         INTEGER NOT NULL CHECK (amount <> 0),
    balance_before INTEGER NOT NULL,
    balance_after  INTEGER NOT NULL CHECK (balance_after >= 0),
    reason         TEXT    NOT NULL,
    created_at     TEXT    NOT NULL,
    CHECK (length(trim(reason)) > 0)
);

CREATE INDEX idx_adjustments_vaccine_month ON inventory_adjustments (vaccine_id, month, created_at);

CREATE TABLE audit_events (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type  TEXT    NOT NULL,
    entity_type TEXT    NOT NULL,
    entity_id   TEXT    NOT NULL DEFAULT '',
    month       TEXT    NOT NULL DEFAULT '',
    reason      TEXT    NOT NULL DEFAULT '',
    metadata    TEXT    NOT NULL DEFAULT '{}',
    created_at  TEXT    NOT NULL
);

CREATE INDEX idx_audit_created ON audit_events (created_at);
CREATE INDEX idx_audit_type ON audit_events (event_type, created_at);
CREATE INDEX idx_audit_month ON audit_events (month, created_at);

CREATE TABLE publications (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    snapshot_id    TEXT    NOT NULL UNIQUE,
    month          TEXT    NOT NULL,
    status         TEXT    NOT NULL CHECK (status IN ('published', 'unpublished')),
    endpoint       TEXT    NOT NULL DEFAULT '',
    version        INTEGER NOT NULL DEFAULT 1,
    payload_sha256 TEXT    NOT NULL DEFAULT '',
    published_at   TEXT    NOT NULL,
    unpublished_at TEXT,
    created_at     TEXT    NOT NULL
);

CREATE INDEX idx_publications_month ON publications (month, created_at);
CREATE INDEX idx_publications_status ON publications (status, published_at);

CREATE TABLE app_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

INSERT INTO app_settings (key, value, updated_at)
VALUES ('low_stock_threshold', '10', strftime('%Y-%m-%dT%H:%M:%SZ', 'now'));
