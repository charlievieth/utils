
CREATE TABLE IF NOT EXISTS command_names (
    `id`          PRIMARY KEY,
    `command`     TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS history (
    `id`          PRIMARY KEY,
    `ppid`        INTEGER NOT NULL,
    `status_code` INTEGER NOT NULL,
    `history_id`  INTEGER NOT NULL, /* TODO: do we need this? */
    `session_id`  TEXT NOT NULL,    /* TODO: use this or the PID? */
    `server_id`   TEXT NOT NULL,    /* TODO: do we need a per-server ID? */
    `username`    TEXT NOT NULL,
    `created_at`  TIMESTAMP NOT NULL,
    `command_id`  INTEGER NOT NULL,
    FOREIGN KEY(command_id) REFERENCES command_names(id)
);


CREATE TABLE IF NOT EXISTS command_arguments (
    `id`          PRIMARY KEY,
    `command_id`  INTEGER NOT NULL,
    `history_id`  INTEGER NOT NULL,
    `argument`    TEXT NOT NULL,
    FOREIGN KEY(history_id) REFERENCES history(id),
    FOREIGN KEY(command_id) REFERENCES command_names(id)
);
