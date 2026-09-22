package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/usememos/memos/store"
)

const testEmbeddingModel = "test-embedding-model"

func TestMemoEmbeddingUpsertReplaces(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	defer ts.Close()
	user, err := createTestingHostUser(ctx, ts)
	require.NoError(t, err)

	memo, err := ts.CreateMemo(ctx, &store.Memo{
		UID:        "embed-memo",
		CreatorID:  user.ID,
		Content:    "memo content",
		Visibility: store.Private,
	})
	require.NoError(t, err)

	// Insert.
	_, err = ts.UpsertMemoEmbedding(ctx, &store.MemoEmbedding{
		MemoID:    memo.ID,
		CreatorID: user.ID,
		Model:     testEmbeddingModel,
		Vector:    []byte{1, 2, 3, 4},
	})
	require.NoError(t, err)

	// Upsert same memo with a new vector and model replaces the row.
	_, err = ts.UpsertMemoEmbedding(ctx, &store.MemoEmbedding{
		MemoID:    memo.ID,
		CreatorID: user.ID,
		Model:     "newer-model",
		Vector:    []byte{5, 6, 7, 8},
	})
	require.NoError(t, err)

	list, err := ts.ListMemoEmbeddings(ctx, &store.FindMemoEmbedding{MemoID: &memo.ID})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "newer-model", list[0].Model)
	require.Equal(t, []byte{5, 6, 7, 8}, list[0].Vector)
	require.NotZero(t, list[0].UpdatedTs)
}

func TestMemoEmbeddingListByCreatorIsolation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	defer ts.Close()
	userA, err := createTestingHostUser(ctx, ts)
	require.NoError(t, err)
	userB, err := createTestingUserWithRole(ctx, ts, "user-b", store.RoleUser)
	require.NoError(t, err)

	memoA, err := ts.CreateMemo(ctx, &store.Memo{
		UID: "memo-a", CreatorID: userA.ID, Content: "a", Visibility: store.Private,
	})
	require.NoError(t, err)
	memoB, err := ts.CreateMemo(ctx, &store.Memo{
		UID: "memo-b", CreatorID: userB.ID, Content: "b", Visibility: store.Private,
	})
	require.NoError(t, err)

	for _, e := range []*store.MemoEmbedding{
		{MemoID: memoA.ID, CreatorID: userA.ID, Model: testEmbeddingModel, Vector: []byte{1}},
		{MemoID: memoB.ID, CreatorID: userB.ID, Model: testEmbeddingModel, Vector: []byte{2}},
	} {
		_, err = ts.UpsertMemoEmbedding(ctx, e)
		require.NoError(t, err)
	}

	// Filtering by creator must return only that creator's rows.
	listA, err := ts.ListMemoEmbeddings(ctx, &store.FindMemoEmbedding{CreatorID: &userA.ID})
	require.NoError(t, err)
	require.Len(t, listA, 1)
	require.Equal(t, memoA.ID, listA[0].MemoID)

	listB, err := ts.ListMemoEmbeddings(ctx, &store.FindMemoEmbedding{CreatorID: &userB.ID})
	require.NoError(t, err)
	require.Len(t, listB, 1)
	require.Equal(t, memoB.ID, listB[0].MemoID)
}

func TestMemoEmbeddingDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	defer ts.Close()
	user, err := createTestingHostUser(ctx, ts)
	require.NoError(t, err)

	memo, err := ts.CreateMemo(ctx, &store.Memo{
		UID: "embed-memo", CreatorID: user.ID, Content: "content", Visibility: store.Private,
	})
	require.NoError(t, err)
	_, err = ts.UpsertMemoEmbedding(ctx, &store.MemoEmbedding{
		MemoID: memo.ID, CreatorID: user.ID, Model: testEmbeddingModel, Vector: []byte{1},
	})
	require.NoError(t, err)

	err = ts.DeleteMemoEmbedding(ctx, &store.DeleteMemoEmbedding{MemoID: memo.ID})
	require.NoError(t, err)

	list, err := ts.ListMemoEmbeddings(ctx, &store.FindMemoEmbedding{MemoID: &memo.ID})
	require.NoError(t, err)
	require.Len(t, list, 0)
}

func TestMemoEmbeddingCascadeOnMemoDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	defer ts.Close()
	user, err := createTestingHostUser(ctx, ts)
	require.NoError(t, err)

	memo, err := ts.CreateMemo(ctx, &store.Memo{
		UID: "embed-memo", CreatorID: user.ID, Content: "content", Visibility: store.Private,
	})
	require.NoError(t, err)
	_, err = ts.UpsertMemoEmbedding(ctx, &store.MemoEmbedding{
		MemoID: memo.ID, CreatorID: user.ID, Model: testEmbeddingModel, Vector: []byte{1},
	})
	require.NoError(t, err)

	err = ts.DeleteMemo(ctx, &store.DeleteMemo{ID: memo.ID})
	require.NoError(t, err)

	list, err := ts.ListMemoEmbeddings(ctx, &store.FindMemoEmbedding{MemoID: &memo.ID})
	require.NoError(t, err)
	require.Len(t, list, 0)
}

func TestListMemosNeedingEmbedding(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ts := NewTestingStore(ctx, t)
	defer ts.Close()
	user, err := createTestingHostUser(ctx, ts)
	require.NoError(t, err)

	// Memo with no embedding: needs one.
	pending, err := ts.CreateMemo(ctx, &store.Memo{
		UID: "pending", CreatorID: user.ID, Content: "pending content", Visibility: store.Private,
	})
	require.NoError(t, err)

	// Memo already embedded for the model: excluded.
	embedded, err := ts.CreateMemo(ctx, &store.Memo{
		UID: "embedded", CreatorID: user.ID, Content: "embedded content", Visibility: store.Private,
	})
	require.NoError(t, err)
	_, err = ts.UpsertMemoEmbedding(ctx, &store.MemoEmbedding{
		MemoID: embedded.ID, CreatorID: user.ID, Model: testEmbeddingModel, Vector: []byte{1},
	})
	require.NoError(t, err)

	// Memo embedded with a DIFFERENT model: still needs the current model.
	staleModel, err := ts.CreateMemo(ctx, &store.Memo{
		UID: "stale-model", CreatorID: user.ID, Content: "stale model content", Visibility: store.Private,
	})
	require.NoError(t, err)
	_, err = ts.UpsertMemoEmbedding(ctx, &store.MemoEmbedding{
		MemoID: staleModel.ID, CreatorID: user.ID, Model: "old-model", Vector: []byte{1},
	})
	require.NoError(t, err)

	// Archived memo: excluded.
	archived, err := ts.CreateMemo(ctx, &store.Memo{
		UID: "archived", CreatorID: user.ID, Content: "archived content", Visibility: store.Private,
	})
	require.NoError(t, err)
	rowStatus := store.Archived
	err = ts.UpdateMemo(ctx, &store.UpdateMemo{ID: archived.ID, RowStatus: &rowStatus})
	require.NoError(t, err)

	memos, err := ts.ListMemosNeedingEmbedding(ctx, testEmbeddingModel, 100)
	require.NoError(t, err)
	ids := map[int32]bool{}
	for _, m := range memos {
		ids[m.ID] = true
		require.NotEmpty(t, m.Content)
		require.Equal(t, user.ID, m.CreatorID)
	}
	require.True(t, ids[pending.ID], "memo without embedding should need one")
	require.True(t, ids[staleModel.ID], "memo embedded with another model should need one")
	require.False(t, ids[embedded.ID], "memo already embedded should be excluded")
	require.False(t, ids[archived.ID], "archived memo should be excluded")

	// Limit is respected.
	limited, err := ts.ListMemosNeedingEmbedding(ctx, testEmbeddingModel, 1)
	require.NoError(t, err)
	require.Len(t, limited, 1)
}
