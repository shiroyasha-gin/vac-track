-- Convert single-vaccine appointments into appointments with one or more
-- vaccine line items. Each item owns its own linked usage entry, so one
-- appointment can consume several vaccines while every dose remains tracked.
CREATE TABLE appointments_backup AS SELECT * FROM appointments;

CREATE TABLE appointments_new (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    patient_id     INTEGER NOT NULL REFERENCES patients (id),
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
    note           TEXT    NOT NULL DEFAULT '',
    status         TEXT    NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'completed', 'cancelled')),
    cancel_reason  TEXT    NOT NULL DEFAULT '',
    completed_at   TEXT,
    cancelled_at   TEXT,
    created_at     TEXT    NOT NULL,
    updated_at     TEXT    NOT NULL,
    CHECK (
        (status = 'completed' AND completed_at IS NOT NULL)
        OR (status <> 'completed' AND completed_at IS NULL)
    ),
    CHECK (
        (status = 'cancelled' AND cancelled_at IS NOT NULL AND length(trim(cancel_reason)) > 0)
        OR (status <> 'cancelled' AND cancelled_at IS NULL AND cancel_reason = '')
    )
);

INSERT INTO appointments_new (
    id, patient_id, scheduled_on, scheduled_time, note, status,
    cancel_reason, completed_at, cancelled_at, created_at, updated_at
)
SELECT
    id, patient_id, scheduled_on, scheduled_time, note, status,
    cancel_reason, completed_at, cancelled_at, created_at, updated_at
FROM appointments;

DROP TABLE appointments;
ALTER TABLE appointments_new RENAME TO appointments;

CREATE TABLE appointment_items (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    appointment_id  INTEGER NOT NULL REFERENCES appointments (id) ON DELETE CASCADE,
    vaccine_id     INTEGER NOT NULL REFERENCES vaccines (id),
    quantity       INTEGER NOT NULL CHECK (quantity > 0),
    usage_entry_id INTEGER REFERENCES usage_entries (id),
    created_at     TEXT    NOT NULL,
    updated_at     TEXT    NOT NULL,
    UNIQUE (appointment_id, vaccine_id)
);

INSERT INTO appointment_items (
    appointment_id, vaccine_id, quantity, usage_entry_id, created_at, updated_at
)
SELECT
    id, vaccine_id, quantity, usage_entry_id, created_at, updated_at
FROM appointments_backup;

DROP TABLE appointments_backup;

CREATE INDEX idx_appointments_patient ON appointments (patient_id, scheduled_on DESC, id DESC);
CREATE INDEX idx_appointments_schedule ON appointments (scheduled_on, scheduled_time, status);
CREATE UNIQUE INDEX idx_appointment_items_usage_unique ON appointment_items (usage_entry_id) WHERE usage_entry_id IS NOT NULL;
CREATE INDEX idx_appointment_items_appointment ON appointment_items (appointment_id, id);
CREATE INDEX idx_appointment_items_vaccine ON appointment_items (vaccine_id);
