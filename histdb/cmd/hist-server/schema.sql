CREATE TABLE IF NOT EXISTS history (
    `id`          PRIMARY KEY,
    `ppid`        INTEGER NOT NULL,
    `status_code` INTEGER NOT NULL,
    `history_id`  INTEGER NOT NULL, /* TODO: do we need this? */
    `session_id`  TEXT NOT NULL,    /* TODO: use this or the PID? */
    `server_id`   TEXT NOT NULL,    /* TODO: do we need a per-server ID? */
    `username`    TEXT NOT NULL,
    `created_at`  TIMESTAMP NOT NULL,
    `command`     TEXT NOT NULL /* Command name */
    `arguments`   TEXT NOT NULL /* JSON encoded argv */
);
