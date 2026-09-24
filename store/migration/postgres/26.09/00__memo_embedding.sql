-- memo_embedding stores one embedding vector per memo for semantic search.
-- vector is little-endian float32s; model records which embedding model produced it
-- so vectors of different models/dimensions are never compared.
-- Idempotent: fork databases created before the calendar baseline already have this table.
-- Rows are removed explicitly by the store when a memo is deleted (SQLite runs without
-- foreign_keys), so no FK is declared and future memo table rebuilds stay safe.
CREATE TABLE IF NOT EXISTS memo_embedding (
  id         SERIAL  PRIMARY KEY,
  memo_id    INTEGER NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,
  model      TEXT    NOT NULL,
  vector     BYTEA   NOT NULL,
  updated_ts BIGINT  NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())
);

CREATE INDEX IF NOT EXISTS idx_memo_embedding_creator_id ON memo_embedding(creator_id);
