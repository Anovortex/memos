package embedding

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/usememos/memos/internal/ai"
	aiembedding "github.com/usememos/memos/internal/ai/embedding"
	"github.com/usememos/memos/internal/plan"
	"github.com/usememos/memos/internal/profile"
	storepb "github.com/usememos/memos/proto/gen/store"
	"github.com/usememos/memos/store"
	teststore "github.com/usememos/memos/store/test"
)

const testModel = "gemini-embedding-001"

type fakeEmbedder struct {
	calls          []aiembedding.Request
	tokensPerBatch int64
}

func (f *fakeEmbedder) Embed(_ context.Context, req aiembedding.Request) (*aiembedding.Response, error) {
	f.calls = append(f.calls, req)
	vectors := make([][]float32, len(req.Input))
	for i := range req.Input {
		vectors[i] = []float32{1, 0}
	}
	return &aiembedding.Response{Vectors: vectors, TokensUsed: f.tokensPerBatch}, nil
}

func newTestRunner(t *testing.T, ts *store.Store, p *profile.Profile, fake *fakeEmbedder) *Runner {
	t.Helper()
	r := NewRunner(ts, p)
	r.newEmbedder = func(ai.ProviderConfig) (aiembedding.Embedder, error) {
		return fake, nil
	}
	return r
}

func createUser(ctx context.Context, t *testing.T, ts *store.Store, username string, role store.Role) *store.User {
	t.Helper()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("test_password"), bcrypt.DefaultCost)
	require.NoError(t, err)
	user, err := ts.CreateUser(ctx, &store.User{
		Username:     username,
		Role:         role,
		Email:        username + "@test.com",
		Nickname:     username,
		PasswordHash: string(passwordHash),
	})
	require.NoError(t, err)
	return user
}

func configureEmbedding(ctx context.Context, t *testing.T, ts *store.Store) {
	t.Helper()
	_, err := ts.UpsertInstanceSetting(ctx, &storepb.InstanceSetting{
		Key: storepb.InstanceSettingKey_AI,
		Value: &storepb.InstanceSetting_AiSetting{
			AiSetting: &storepb.InstanceAISetting{
				Providers: []*storepb.AIProviderConfig{{
					Id:     "gp",
					Title:  "Gemini",
					Type:   storepb.AIProviderType_GEMINI,
					ApiKey: "test-key",
				}},
				Embedding: &storepb.EmbeddingConfig{ProviderId: "gp", Model: testModel},
			},
		},
	})
	require.NoError(t, err)
}

func TestRunOnceEmbedsPendingMemos(t *testing.T) {
	ctx := context.Background()
	ts := teststore.NewTestingStore(ctx, t)
	defer ts.Close()
	user := createUser(ctx, t, ts, "host", store.RoleAdmin)
	configureEmbedding(ctx, t, ts)

	memo, err := ts.CreateMemo(ctx, &store.Memo{
		UID: "m1", CreatorID: user.ID, Content: "kubernetes ingress debugging", Visibility: store.Private,
	})
	require.NoError(t, err)

	fake := &fakeEmbedder{tokensPerBatch: 12}
	runner := newTestRunner(t, ts, &profile.Profile{}, fake)
	runner.RunOnce(ctx)

	rows, err := ts.ListMemoEmbeddings(ctx, &store.FindMemoEmbedding{MemoID: &memo.ID})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, user.ID, rows[0].CreatorID)
	require.Equal(t, testModel, rows[0].Model)
	require.Len(t, fake.calls, 1)
	require.Equal(t, aiembedding.TaskDocument, fake.calls[0].TaskType)

	// Usage was recorded.
	used, err := ts.GetUserAIUsage(ctx, user.ID, plan.UsageDate(time.Now()))
	require.NoError(t, err)
	require.Equal(t, int64(12), used)

	// Second run has nothing left to do.
	runner.RunOnce(ctx)
	require.Len(t, fake.calls, 1)
}

func TestRunOnceNoOpWhenUnconfigured(t *testing.T) {
	ctx := context.Background()
	ts := teststore.NewTestingStore(ctx, t)
	defer ts.Close()
	user := createUser(ctx, t, ts, "host", store.RoleAdmin)

	_, err := ts.CreateMemo(ctx, &store.Memo{
		UID: "m1", CreatorID: user.ID, Content: "content", Visibility: store.Private,
	})
	require.NoError(t, err)

	fake := &fakeEmbedder{}
	runner := newTestRunner(t, ts, &profile.Profile{}, fake)
	runner.RunOnce(ctx)

	require.Len(t, fake.calls, 0)
}

func TestRunOnceSkipsCappedUserButProcessesOthers(t *testing.T) {
	ctx := context.Background()
	ts := teststore.NewTestingStore(ctx, t)
	defer ts.Close()
	capped := createUser(ctx, t, ts, "capped", store.RoleAdmin)
	fresh := createUser(ctx, t, ts, "fresh", store.RoleUser)
	configureEmbedding(ctx, t, ts)

	cappedMemo, err := ts.CreateMemo(ctx, &store.Memo{
		UID: "capped-memo", CreatorID: capped.ID, Content: "capped content", Visibility: store.Private,
	})
	require.NoError(t, err)
	freshMemo, err := ts.CreateMemo(ctx, &store.Memo{
		UID: "fresh-memo", CreatorID: fresh.ID, Content: "fresh content", Visibility: store.Private,
	})
	require.NoError(t, err)

	// The capped user has already spent the daily budget.
	require.NoError(t, ts.IncrementUserAIUsage(ctx, capped.ID, plan.UsageDate(time.Now()), 100))

	fake := &fakeEmbedder{tokensPerBatch: 5}
	runner := newTestRunner(t, ts, &profile.Profile{FreeTierAITokensPerDay: 100}, fake)
	runner.RunOnce(ctx)

	cappedRows, err := ts.ListMemoEmbeddings(ctx, &store.FindMemoEmbedding{MemoID: &cappedMemo.ID})
	require.NoError(t, err)
	require.Len(t, cappedRows, 0, "capped user's memo must not be embedded")

	freshRows, err := ts.ListMemoEmbeddings(ctx, &store.FindMemoEmbedding{MemoID: &freshMemo.ID})
	require.NoError(t, err)
	require.Len(t, freshRows, 1, "uncapped user's memo must be embedded")
}

func TestRunOnceTerminatesWhenAllRemainingCapped(t *testing.T) {
	ctx := context.Background()
	ts := teststore.NewTestingStore(ctx, t)
	defer ts.Close()
	user := createUser(ctx, t, ts, "host", store.RoleAdmin)
	configureEmbedding(ctx, t, ts)

	_, err := ts.CreateMemo(ctx, &store.Memo{
		UID: "m1", CreatorID: user.ID, Content: "content", Visibility: store.Private,
	})
	require.NoError(t, err)
	require.NoError(t, ts.IncrementUserAIUsage(ctx, user.ID, plan.UsageDate(time.Now()), 100))

	fake := &fakeEmbedder{}
	runner := newTestRunner(t, ts, &profile.Profile{FreeTierAITokensPerDay: 100}, fake)

	done := make(chan struct{})
	go func() {
		runner.RunOnce(ctx)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("RunOnce did not terminate with all users capped")
	}
	require.Len(t, fake.calls, 0)
}

func TestReEmbedAfterEmbeddingDeleted(t *testing.T) {
	ctx := context.Background()
	ts := teststore.NewTestingStore(ctx, t)
	defer ts.Close()
	user := createUser(ctx, t, ts, "host", store.RoleAdmin)
	configureEmbedding(ctx, t, ts)

	memo, err := ts.CreateMemo(ctx, &store.Memo{
		UID: "m1", CreatorID: user.ID, Content: "original", Visibility: store.Private,
	})
	require.NoError(t, err)

	fake := &fakeEmbedder{tokensPerBatch: 1}
	runner := newTestRunner(t, ts, &profile.Profile{}, fake)
	runner.RunOnce(ctx)
	require.Len(t, fake.calls, 1)

	// Content update path deletes the row (done by the API layer), then pokes.
	require.NoError(t, ts.DeleteMemoEmbedding(ctx, &store.DeleteMemoEmbedding{MemoID: memo.ID}))
	runner.RunOnce(ctx)
	require.Len(t, fake.calls, 2)

	rows, err := ts.ListMemoEmbeddings(ctx, &store.FindMemoEmbedding{MemoID: &memo.ID})
	require.NoError(t, err)
	require.Len(t, rows, 1)
}
