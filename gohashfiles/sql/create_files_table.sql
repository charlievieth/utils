CREATE TABLE IF NOT EXISTS files (
    id          INTEGER PRIMARY KEY,
    path        TEXT NOT NULL UNIQUE,
    basename    TEXT NOT NULL,
    extname     TEXT NOT NULL,
    hash        TEXT NOT NULL,
    size        INTEGER NOT NULL
);
