BEGIN;

CREATE TABLE IF NOT EXISTS run_ids (
    `id` INTEGER PRIMARY KEY
);

-- Make sure 0 is never a valid value
INSERT OR IGNORE INTO run_ids(id) VALUES (0);

COMMIT;
