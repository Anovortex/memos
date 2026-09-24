package v1

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/usememos/memos/internal/ai/embedding"
	"github.com/usememos/memos/internal/plan"
	"github.com/usememos/memos/internal/ratelimit"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	embeddingrunner "github.com/usememos/memos/server/runner/embedding"
	"github.com/usememos/memos/store"
)

const (
	defaultSearchLimit = 20
	maxSearchLimit     = 50
	// maxSearchQueryRunes bounds the embedded query text.
	maxSearchQueryRunes = 1000
)

// SearchMemos ranks the calling user's own memos by semantic similarity to the
// query. Scope is hard-limited to the requesting user's memos: embeddings are
// listed by creator and the re-fetch filters by creator again.
func (s *APIV1Service) SearchMemos(ctx context.Context, request *v1pb.SearchMemosRequest) (*v1pb.SearchMemosResponse, error) {
	user, err := s.fetchCurrentUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user")
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	if err := s.throttleAndCharge(ratelimit.ScopeSearchUser, userKey(user.ID), 1); err != nil {
		return nil, err
	}
	verified, err := s.Store.IsEmailVerified(ctx, user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check email verification")
	}
	if !verified {
		// PermissionDenied, not FailedPrecondition: the UI treats the latter as
		// "search not configured" and silently falls back to substring search.
		return nil, status.Errorf(codes.PermissionDenied, "verify your email address to use AI features")
	}

	query := strings.TrimSpace(request.GetQuery())
	if query == "" {
		return nil, status.Errorf(codes.InvalidArgument, "query is required")
	}
	limit := int(request.GetLimit())
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	// Daily token cap. Check-then-increment is racy under concurrency; accepted
	// at free-tier scale.
	limits := plan.ForUser(s.Profile, user)
	usageDate := plan.UsageDate(time.Now())
	if limits.AITokensPerDay > 0 {
		used, err := s.Store.GetUserAIUsage(ctx, user.ID, usageDate)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get AI usage")
		}
		if used >= limits.AITokensPerDay {
			return nil, status.Errorf(codes.ResourceExhausted, "daily AI token limit reached (max %d)", limits.AITokensPerDay)
		}
	}

	aiSetting, err := s.Store.GetInstanceAISetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get AI setting")
	}
	provider, model, ok := embeddingrunner.ResolveProvider(aiSetting)
	if !ok {
		return nil, status.Errorf(codes.FailedPrecondition, "semantic search is not configured")
	}
	factory := s.EmbedderFactory
	if factory == nil {
		factory = embeddingrunner.NewEmbedder
	}
	embedder, err := factory(provider)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to build embedder: %v", err)
	}

	queryRunes := []rune(query)
	if len(queryRunes) > maxSearchQueryRunes {
		query = string(queryRunes[:maxSearchQueryRunes])
	}
	embedResp, err := embedder.Embed(ctx, embedding.Request{
		Input:    []string{query},
		Model:    model,
		TaskType: embedding.TaskQuery,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to embed query: %v", err)
	}
	if len(embedResp.Vectors) != 1 {
		return nil, status.Errorf(codes.Internal, "embedder returned %d vectors for the query", len(embedResp.Vectors))
	}
	// Fail closed: a request whose spend cannot be recorded must not succeed,
	// or a persistently failing write would disable the cap entirely.
	if err := s.Store.IncrementUserAIUsage(ctx, user.ID, usageDate, embedResp.TokensUsed); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to record AI usage")
	}

	rows, err := s.Store.ListMemoEmbeddings(ctx, &store.FindMemoEmbedding{
		CreatorID: &user.ID,
		Model:     &model,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list embeddings")
	}
	scored := semanticRank(embedResp.Vectors[0], rows)
	if len(scored) > limit {
		scored = scored[:limit]
	}
	if len(scored) == 0 {
		return &v1pb.SearchMemosResponse{Results: []*v1pb.SearchMemosResponse_Result{}}, nil
	}

	ids := make([]int32, 0, len(scored))
	for _, sm := range scored {
		ids = append(ids, sm.memoID)
	}
	normal := store.Normal
	// CreatorID again is defense in depth on top of the creator-scoped
	// embedding rows; RowStatus excludes memos archived after indexing.
	memos, err := s.Store.ListMemos(ctx, &store.FindMemo{
		IDList:    ids,
		CreatorID: &user.ID,
		RowStatus: &normal,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list memos")
	}
	byID := make(map[int32]*store.Memo, len(memos))
	for _, memo := range memos {
		byID[memo.ID] = memo
	}

	// IDList does not preserve order; rebuild in score order.
	results := make([]*v1pb.SearchMemosResponse_Result, 0, len(scored))
	for _, sm := range scored {
		memo, ok := byID[sm.memoID]
		if !ok {
			continue
		}
		memoMessage, err := s.convertMemoFromStore(ctx, memo, nil, nil, nil)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert memo")
		}
		results = append(results, &v1pb.SearchMemosResponse_Result{
			Memo:  memoMessage,
			Score: sm.score,
		})
	}
	return &v1pb.SearchMemosResponse{Results: results}, nil
}

type scoredMemo struct {
	memoID int32
	score  float32
}

// semanticRank scores rows against the query vector, highest first.
// ponytail: brute-force cosine over one user's vectors; replace with a
// pgvector-backed driver method when Postgres is committed for production.
func semanticRank(queryVector []float32, rows []*store.MemoEmbedding) []scoredMemo {
	scored := make([]scoredMemo, 0, len(rows))
	for _, row := range rows {
		vector, err := embedding.DecodeVector(row.Vector)
		if err != nil {
			slog.Warn("Skipping undecodable memo embedding", slog.Int("memoID", int(row.MemoID)), slog.Any("err", err))
			continue
		}
		scored = append(scored, scoredMemo{
			memoID: row.MemoID,
			score:  embedding.CosineSimilarity(queryVector, vector),
		})
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
	return scored
}
