package store

import (
	"context"
	"time"
)

// MemoEmbedding is one embedding vector for a memo, produced by a specific model.
// Vector holds little-endian float32s; rows for different models are never compared.
type MemoEmbedding struct {
	MemoID    int32
	CreatorID int32
	Model     string
	Vector    []byte
	UpdatedTs int64
}

// FindMemoEmbedding filters embedding rows in list queries.
type FindMemoEmbedding struct {
	MemoID    *int32
	CreatorID *int32
	Model     *string
}

// DeleteMemoEmbedding identifies the embedding row of a memo to remove.
type DeleteMemoEmbedding struct {
	MemoID int32
}

// UpsertMemoEmbedding inserts or replaces the embedding row for a memo.
func (s *Store) UpsertMemoEmbedding(ctx context.Context, upsert *MemoEmbedding) (*MemoEmbedding, error) {
	if upsert.UpdatedTs == 0 {
		upsert.UpdatedTs = time.Now().Unix()
	}
	return s.driver.UpsertMemoEmbedding(ctx, upsert)
}

// ListMemoEmbeddings returns embedding rows matching the filter.
func (s *Store) ListMemoEmbeddings(ctx context.Context, find *FindMemoEmbedding) ([]*MemoEmbedding, error) {
	return s.driver.ListMemoEmbeddings(ctx, find)
}

// DeleteMemoEmbedding removes the embedding row for a memo.
func (s *Store) DeleteMemoEmbedding(ctx context.Context, delete *DeleteMemoEmbedding) error {
	return s.driver.DeleteMemoEmbedding(ctx, delete)
}

// ListMemosNeedingEmbedding returns NORMAL memos that have no embedding row for
// the given model, oldest first. Only ID, CreatorID, and Content are populated.
func (s *Store) ListMemosNeedingEmbedding(ctx context.Context, model string, limit int) ([]*Memo, error) {
	return s.driver.ListMemosNeedingEmbedding(ctx, model, limit)
}
