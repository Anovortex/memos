-- memo_embedding stores one embedding vector per memo for semantic search.
-- vector is little-endian float32s; model records which embedding model produced it
-- so vectors of different models/dimensions are never compared.
-- Idempotent: fork databases created before the calendar baseline already have this table.
-- Rows are removed explicitly by the store when a memo is deleted (SQLite runs without
-- foreign_keys), so no FK is declared and future memo table rebuilds stay safe.
-- MySQL has no CREATE INDEX IF NOT EXISTS, so the index is declared inline.
CREATE TABLE IF NOT EXISTS `memo_embedding` (
  `id`         INT          NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `memo_id`    INT          NOT NULL UNIQUE,
  `creator_id` INT          NOT NULL,
  `model`      VARCHAR(255) NOT NULL,
  `vector`     MEDIUMBLOB   NOT NULL,
  `updated_ts` BIGINT       NOT NULL DEFAULT (UNIX_TIMESTAMP()),
  INDEX `idx_memo_embedding_creator_id` (`creator_id`)
);
