-- +migrate Up
CREATE TABLE epochs(
    id          serial      primary key,
    start_time  timestamp   not null,
    end_time    timestamp   not null  -- update when new epoch started
);

CREATE TABLE party
(
    epoch_id    serial,
    address     text,
    certificate text,
    domen       text,
    FOREIGN KEY (epoch_id) REFERENCES epochs(id),
    primary key (epoch_id, address)
);

CREATE TABLE IF NOT EXISTS latest_block
(
    id INT NOT NULL UNIQUE,
    latest_block_id INT NOT NULL
);


CREATE TABLE IF NOT EXISTS tasks
(
    id          SERIAL PRIMARY KEY,
    task_type   VARCHAR(64) NOT NULL,
    status      INT NOT NULL,
    data        TEXT NOT NULL,
    error       TEXT,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_created_at ON tasks(created_at);

-- +migrate Down
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS latest_block;
DROP TABLE IF EXISTS party;
DROP TABLE IF EXISTS epochs;

DROP INDEX IF EXISTS idx_tasks_status;
DROP INDEX IF EXISTS idx_tasks_created_at;
