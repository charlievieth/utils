BEGIN;

CREATE TABLE IF NOT EXISTS audio_files (
    id       INTEGER PRIMARY KEY,
    filename TEXT NOT NULL,
    hash     TEXT NOT NULL,
    size     INTEGER NOT NULL,
    modtime  DATETIME NOT NULL,
    UNIQUE(filename)
);

COMMIT;
