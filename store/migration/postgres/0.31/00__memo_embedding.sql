-- memo_embedding stores one embedding vector per memo for semantic search.
-- vector is little-endian float32s; model records which embedding model produced it
-- so vectors of different models/dimensions are never compared.
-- ON DELETE CASCADE cleans up when the parent memo is deleted.
CREATE TABLE memo_embedding (
  id         SERIAL  PRIMARY KEY,
  memo_id    INTEGER NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,
  model      TEXT    NOT NULL,
  vector     BYTEA   NOT NULL,
  updated_ts BIGINT  NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  FOREIGN KEY (memo_id) REFERENCES memo(id) ON DELETE CASCADE
);

CREATE INDEX idx_memo_embedding_creator_id ON memo_embedding(creator_id);
