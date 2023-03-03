-- migrate:up

CREATE TABLE devices (
    id VARCHAR(64) PRIMARY KEY,
    type VARCHAR(32) NOT NULL,
    owner VARCHAR(64) NOT NULL,
    allow BOOLEAN NOT NULL,
    data TEXT,
    created_at DATETIME
);

-- migrate:down

DROP TABLE devices;