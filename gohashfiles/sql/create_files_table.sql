BEGIN;

CREATE TABLE IF NOT EXISTS files (
    id          INTEGER PRIMARY KEY,
    run_id      INTEGER NOT NULL,
    hash        TEXT NOT NULL,
    path        TEXT NOT NULL UNIQUE,
    basename    TEXT NOT NULL,
    extname     TEXT,
    size        INTEGER NOT NULL,
    FOREIGN KEY(run_id) REFERENCES run_ids(id)
);

COMMIT;
