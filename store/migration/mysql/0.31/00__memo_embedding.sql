-- memo_embedding stores one embedding vector per memo for semantic search.
-- vector is little-endian float32s; model records which embedding model produced it
-- so vectors of different models/dimensions are never compared.
-- ON DELETE CASCADE cleans up when the parent memo is deleted.
CREATE TABLE `memo_embedding` (
  `id`         INT          NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `memo_id`    INT          NOT NULL UNIQUE,
  `creator_id` INT          NOT NULL,
  `model`      VARCHAR(255) NOT NULL,
  `vector`     MEDIUMBLOB   NOT NULL,
  `updated_ts` BIGINT       NOT NULL DEFAULT (UNIX_TIMESTAMP()),
  FOREIGN KEY (`memo_id`) REFERENCES `memo`(`id`) ON DELETE CASCADE
);

CREATE INDEX `idx_memo_embedding_creator_id` ON `memo_embedding`(`creator_id`);
