CREATE TABLE IF NOT EXISTS "schema_migrations" (version varchar(255) primary key);
CREATE TABLE devices (
    id VARCHAR(64) PRIMARY KEY,
    type VARCHAR(32) NOT NULL,
    owner VARCHAR(64) NOT NULL,
    allow BOOLEAN NOT NULL,
    data TEXT,
    created_at DATETIME
);
-- Dbmate schema migrations
INSERT INTO "schema_migrations" (version) VALUES
  ('20230302224007');
