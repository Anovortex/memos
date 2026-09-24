package test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	aiembedding "github.com/usememos/memos/internal/ai/embedding"
	"github.com/usememos/memos/internal/plan"
	apiv1 "github.com/usememos/memos/proto/gen/api/v1"
	storepb "github.com/usememos/memos/proto/gen/store"
	"github.com/usememos/memos/provider/ai"
	"github.com/usememos/memos/store"
)

const searchTestModel = "gemini-embedding-001"

// fixedQueryEmbedder returns the same query vector for every request; ranking
// then depends only on the vectors stored per memo.
type fixedQueryEmbedder struct {
	queryVector []float32
	tokens      int64
	calls       int
}

func (f *fixedQueryEmbedder) Embed(_ context.Context, req aiembedding.Request) (*aiembedding.Response, error) {
	f.calls++
	vectors := make([][]float32, len(req.Input))
	for i := range req.Input {
		vectors[i] = f.queryVector
	}
	return &aiembedding.Response{Vectors: vectors, TokensUsed: f.tokens}, nil
}

func configureSearch(ctx context.Context, t *testing.T, ts *TestService, fake *fixedQueryEmbedder) {
	t.Helper()
	_, err := ts.Store.UpsertInstanceSetting(ctx, &storepb.InstanceSetting{
		Key: storepb.InstanceSettingKey_AI,
		Value: &storepb.InstanceSetting_AiSetting{
			AiSetting: &storepb.InstanceAISetting{
				Providers: []*storepb.AIProviderConfig{{
					Id:     "gp",
					Title:  "Gemini",
					Type:   storepb.AIProviderType_GEMINI,
					ApiKey: "test-key",
				}},
				Embedding: &storepb.EmbeddingConfig{ProviderId: "gp", Model: searchTestModel},
			},
		},
	})
	require.NoError(t, err)
	ts.Service.EmbedderFactory = func(ai.ProviderConfig) (aiembedding.Embedder, error) {
		return fake, nil
	}
}

func createPrivateMemoWithVector(ctx context.Context, t *testing.T, ts *TestService, user *store.User, uid, content string, vector []float32) *store.Memo {
	t.Helper()
	memo, err := ts.Store.CreateMemo(ctx, &store.Memo{
		UID:        uid,
		CreatorID:  user.ID,
		Content:    content,
		Visibility: store.Private,
	})
	require.NoError(t, err)
	_, err = ts.Store.UpsertMemoEmbedding(ctx, &store.MemoEmbedding{
		MemoID:    memo.ID,
		CreatorID: user.ID,
		Model:     searchTestModel,
		Vector:    aiembedding.EncodeVector(vector),
	})
	require.NoError(t, err)
	return memo
}

func TestSearchMemosIsolationBetweenUsers(t *testing.T) {
	// Arrange: A's private memo matches the query perfectly; B's does not.
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	userA, err := ts.CreateHostUser(ctx, "user-a")
	require.NoError(t, err)
	userB, err := ts.CreateRegularUser(ctx, "user-b")
	require.NoError(t, err)
	fake := &fixedQueryEmbedder{queryVector: []float32{1, 0}, tokens: 3}
	configureSearch(ctx, t, ts, fake)

	memoA := createPrivateMemoWithVector(ctx, t, ts, userA, "a-secret", "user A private secret", []float32{1, 0})
	memoB := createPrivateMemoWithVector(ctx, t, ts, userB, "b-note", "user B unrelated note", []float32{0, 1})

	// Act: B searches; A's vector is the best match to the query.
	resp, err := ts.Service.SearchMemos(ts.CreateUserContext(ctx, userB.ID), &apiv1.SearchMemosRequest{Query: "secret"})

	// Assert: only B's own memo comes back, never A's.
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	require.Contains(t, resp.Results[0].Memo.Name, memoB.UID)
	for _, result := range resp.Results {
		require.NotContains(t, result.Memo.Name, memoA.UID, "search must never surface another user's memo")
	}

	// And A finds A's memo.
	respA, err := ts.Service.SearchMemos(ts.CreateUserContext(ctx, userA.ID), &apiv1.SearchMemosRequest{Query: "secret"})
	require.NoError(t, err)
	require.Len(t, respA.Results, 1)
	require.Contains(t, respA.Results[0].Memo.Name, memoA.UID)
}

func TestSearchMemosOrderedByScore(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	user, err := ts.CreateHostUser(ctx, "host")
	require.NoError(t, err)
	fake := &fixedQueryEmbedder{queryVector: []float32{1, 0}, tokens: 1}
	configureSearch(ctx, t, ts, fake)

	far := createPrivateMemoWithVector(ctx, t, ts, user, "far", "far memo", []float32{0, 1})
	near := createPrivateMemoWithVector(ctx, t, ts, user, "near", "near memo", []float32{1, 0})
	mid := createPrivateMemoWithVector(ctx, t, ts, user, "mid", "mid memo", []float32{1, 1})

	resp, err := ts.Service.SearchMemos(ts.CreateUserContext(ctx, user.ID), &apiv1.SearchMemosRequest{Query: "q"})
	require.NoError(t, err)
	require.Len(t, resp.Results, 3)
	require.Contains(t, resp.Results[0].Memo.Name, near.UID)
	require.Contains(t, resp.Results[1].Memo.Name, mid.UID)
	require.Contains(t, resp.Results[2].Memo.Name, far.UID)
	require.GreaterOrEqual(t, resp.Results[0].Score, resp.Results[1].Score)
	require.GreaterOrEqual(t, resp.Results[1].Score, resp.Results[2].Score)
}

func TestSearchMemosLimitClamp(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	user, err := ts.CreateHostUser(ctx, "host")
	require.NoError(t, err)
	fake := &fixedQueryEmbedder{queryVector: []float32{1, 0}, tokens: 1}
	configureSearch(ctx, t, ts, fake)

	for i := 0; i < 3; i++ {
		createPrivateMemoWithVector(ctx, t, ts, user, "memo-"+string(rune('a'+i)), "content", []float32{1, 0})
	}

	resp, err := ts.Service.SearchMemos(ts.CreateUserContext(ctx, user.ID), &apiv1.SearchMemosRequest{Query: "q", Limit: 2})
	require.NoError(t, err)
	require.Len(t, resp.Results, 2)
}

func TestSearchMemosExcludesArchived(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	user, err := ts.CreateHostUser(ctx, "host")
	require.NoError(t, err)
	fake := &fixedQueryEmbedder{queryVector: []float32{1, 0}, tokens: 1}
	configureSearch(ctx, t, ts, fake)

	memo := createPrivateMemoWithVector(ctx, t, ts, user, "to-archive", "content", []float32{1, 0})
	archived := store.Archived
	require.NoError(t, ts.Store.UpdateMemo(ctx, &store.UpdateMemo{ID: memo.ID, RowStatus: &archived}))

	resp, err := ts.Service.SearchMemos(ts.CreateUserContext(ctx, user.ID), &apiv1.SearchMemosRequest{Query: "q"})
	require.NoError(t, err)
	require.Len(t, resp.Results, 0)
}

func TestSearchMemosUnauthenticated(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	fake := &fixedQueryEmbedder{queryVector: []float32{1, 0}}
	configureSearch(ctx, t, ts, fake)

	_, err := ts.Service.SearchMemos(ctx, &apiv1.SearchMemosRequest{Query: "q"})
	require.Error(t, err)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestSearchMemosUnconfigured(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	user, err := ts.CreateHostUser(ctx, "host")
	require.NoError(t, err)

	_, err = ts.Service.SearchMemos(ts.CreateUserContext(ctx, user.ID), &apiv1.SearchMemosRequest{Query: "q"})
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestSearchMemosMetersTokensAndEnforcesCap(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	user, err := ts.CreateHostUser(ctx, "host")
	require.NoError(t, err)
	fake := &fixedQueryEmbedder{queryVector: []float32{1, 0}, tokens: 60}
	configureSearch(ctx, t, ts, fake)
	ts.Profile.FreeTierAITokensPerDay = 100

	userCtx := ts.CreateUserContext(ctx, user.ID)

	// First search is allowed and metered.
	_, err = ts.Service.SearchMemos(userCtx, &apiv1.SearchMemosRequest{Query: "q"})
	require.NoError(t, err)
	used, err := ts.Store.GetUserAIUsage(ctx, user.ID, plan.UsageDate(time.Now()))
	require.NoError(t, err)
	require.Equal(t, int64(60), used)

	// Second search is allowed (60 < 100), pushing usage past the cap.
	_, err = ts.Service.SearchMemos(userCtx, &apiv1.SearchMemosRequest{Query: "q"})
	require.NoError(t, err)

	// Third search is rejected.
	_, err = ts.Service.SearchMemos(userCtx, &apiv1.SearchMemosRequest{Query: "q"})
	require.Error(t, err)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	require.Equal(t, 2, fake.calls)
}

func TestCreateMemoCountCap(t *testing.T) {
	ctx := context.Background()
	ts := NewTestService(t)
	defer ts.Cleanup()
	user, err := ts.CreateHostUser(ctx, "host")
	require.NoError(t, err)
	ts.Profile.FreeTierMaxMemos = 2
	userCtx := ts.CreateUserContext(ctx, user.ID)

	first, err := ts.Service.CreateMemo(userCtx, &apiv1.CreateMemoRequest{
		Memo: &apiv1.Memo{Content: "one", Visibility: apiv1.Visibility_PRIVATE},
	})
	require.NoError(t, err)
	_, err = ts.Service.CreateMemo(userCtx, &apiv1.CreateMemoRequest{
		Memo: &apiv1.Memo{Content: "two", Visibility: apiv1.Visibility_PRIVATE},
	})
	require.NoError(t, err)

	// Third memo hits the cap.
	_, err = ts.Service.CreateMemo(userCtx, &apiv1.CreateMemoRequest{
		Memo: &apiv1.Memo{Content: "three", Visibility: apiv1.Visibility_PRIVATE},
	})
	require.Error(t, err)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))

	// Comments count toward the cap too.
	_, err = ts.Service.CreateMemoComment(userCtx, &apiv1.CreateMemoCommentRequest{
		Name: first.Name,
		Comment: &apiv1.Memo{
			Content:    "comment at cap",
			Visibility: apiv1.Visibility_PRIVATE,
		},
	})
	require.Error(t, err)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))

	// Zero means unlimited.
	ts.Profile.FreeTierMaxMemos = 0
	_, err = ts.Service.CreateMemo(userCtx, &apiv1.CreateMemoRequest{
		Memo: &apiv1.Memo{Content: "unlimited again", Visibility: apiv1.Visibility_PRIVATE},
	})
	require.NoError(t, err)
}
