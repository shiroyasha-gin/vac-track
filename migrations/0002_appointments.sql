-- Appointments are local scheduling records. A completed appointment owns
-- exactly one usage entry; the usage entry is what deducts stock and appears in
-- monthly reports. Cancelled appointments never create usage.
CREATE TABLE appointments (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    patient_id     INTEGER NOT NULL REFERENCES patients (id),
    vaccine_id     INTEGER NOT NULL REFERENCES vaccines (id),
    scheduled_on   TEXT    NOT NULL CHECK (
        length(scheduled_on) = 10 AND scheduled_on GLOB '[0-9][0-9][0-9][0-9]-[0-1][0-9]-[0-3][0-9]'
    ),
    scheduled_time TEXT    NOT NULL DEFAULT '' CHECK (
        scheduled_time = '' OR (
            length(scheduled_time) = 5
            AND scheduled_time GLOB '[0-2][0-9]:[0-5][0-9]'
            AND substr(scheduled_time, 1, 2) <= '23'
        )
    ),
    quantity       INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
    note           TEXT    NOT NULL DEFAULT '',
    status         TEXT    NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'completed', 'cancelled')),
    cancel_reason  TEXT    NOT NULL DEFAULT '',
    usage_entry_id INTEGER REFERENCES usage_entries (id),
    completed_at   TEXT,
    cancelled_at   TEXT,
    created_at     TEXT    NOT NULL,
    updated_at     TEXT    NOT NULL,
    CHECK (
        (status = 'completed' AND usage_entry_id IS NOT NULL AND completed_at IS NOT NULL)
        OR (status <> 'completed' AND usage_entry_id IS NULL AND completed_at IS NULL)
    ),
    CHECK (
        (status = 'cancelled' AND cancelled_at IS NOT NULL AND length(trim(cancel_reason)) > 0)
        OR (status <> 'cancelled' AND cancelled_at IS NULL AND cancel_reason = '')
    )
);

CREATE INDEX idx_appointments_patient ON appointments (patient_id, scheduled_on DESC, id DESC);
CREATE INDEX idx_appointments_schedule ON appointments (scheduled_on, scheduled_time, status);
CREATE UNIQUE INDEX idx_appointments_usage_unique ON appointments (usage_entry_id) WHERE usage_entry_id IS NOT NULL;
