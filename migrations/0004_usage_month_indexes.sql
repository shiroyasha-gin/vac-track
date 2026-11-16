-- Month-wide reports scan every usage entry for one month. The original
-- indexes all started with vaccine_id or patient_id, so month totals, daily
-- trends and monthly exports fell back to a full table scan.
CREATE INDEX idx_usage_active_month
    ON usage_entries (month, vaccine_id, administered_on)
    WHERE voided_at IS NULL;

-- Exports keep voided entries for audit context and order by administered_on,
-- so they need the month prefix without the active-only predicate.
CREATE INDEX idx_usage_month
    ON usage_entries (month, administered_on);
