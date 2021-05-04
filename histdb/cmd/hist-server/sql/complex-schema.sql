CREATE TABLE IF NOT EXISTS session_ids (
    `id` INTEGER PRIMARY KEY
);

-- Create new session command
-- INSERT INTO sessions DEFAULT VALUES;

CREATE TABLE IF NOT EXISTS command_names (
    `id`          INTEGER PRIMARY KEY,
    `command`     TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS history (
    `id`          INTEGER PRIMARY KEY,
    `ppid`        INTEGER NOT NULL,
    `status_code` INTEGER NOT NULL,
    `history_id`  INTEGER NOT NULL, /* TODO: do we need this? */
    `session_id`  INTEGER NOT NULL, /* TODO: use this or the PID? */
    `server_id`   TEXT NOT NULL,    /* TODO: do we need a per-server ID? */
    `username`    TEXT NOT NULL,
    `created_at`  TIMESTAMP NOT NULL,
    `command_id`  INTEGER NOT NULL,
    FOREIGN KEY(command_id) REFERENCES command_names(id),
    FOREIGN KEY(session_id) REFERENCES session_ids(id)
);


CREATE TABLE IF NOT EXISTS command_arguments (
    `id`          PRIMARY KEY,
    `command_id`  INTEGER NOT NULL,
    `history_id`  INTEGER NOT NULL,
    `argument`    TEXT NOT NULL,
    FOREIGN KEY(history_id) REFERENCES history(id),
    FOREIGN KEY(command_id) REFERENCES command_names(id)
);
