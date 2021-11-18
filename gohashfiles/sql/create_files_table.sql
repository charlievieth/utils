CREATE TABLE IF NOT EXISTS files (
    id          INTEGER PRIMARY KEY,
    hash        TEXT NOT NULL,
    path        TEXT NOT NULL UNIQUE,
    basename    TEXT NOT NULL,
    extname     TEXT,
    size        INTEGER NOT NULL
);
