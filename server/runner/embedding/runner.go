// Package embedding is the background indexer for semantic search: it embeds
// memos that have no embedding row for the configured model, per user, within
// each user's daily AI token cap.
package embedding

import (
	"context"
	"log/slog"
	"time"

	"github.com/pkg/errors"

	"github.com/usememos/memos/internal/ai"
	aiembedding "github.com/usememos/memos/internal/ai/embedding"
	"github.com/usememos/memos/internal/ai/embedding/gemini"
	"github.com/usememos/memos/internal/plan"
	"github.com/usememos/memos/internal/profile"
	storepb "github.com/usememos/memos/proto/gen/store"
	"github.com/usememos/memos/store"
)

const (
	// runnerInterval also serves as the retry cadence after provider errors
	// (e.g. free-tier 429s): failed memos stay unembedded and are picked up by
	// the next missing-row scan.
	runnerInterval = 5 * time.Minute
	batchSize      = 100
	// maxEmbedRunes bounds one memo's embedding input; longer content is truncated.
	maxEmbedRunes = 8000
)

// Runner embeds memos in the background.
type Runner struct {
	store   *store.Store
	profile *profile.Profile
	poke    chan struct{}
	// newEmbedder is a test seam; production uses provider-type dispatch.
	newEmbedder func(cfg ai.ProviderConfig) (aiembedding.Embedder, error)
}

// NewRunner constructs a Runner.
func NewRunner(s *store.Store, p *profile.Profile) *Runner {
	return &Runner{
		store:       s,
		profile:     p,
		poke:        make(chan struct{}, 1),
		newEmbedder: NewEmbedder,
	}
}

// Poke asks the runner to index soon without blocking the caller.
func (r *Runner) Poke() {
	select {
	case r.poke <- struct{}{}:
	default:
	}
}

// Run indexes on a fixed interval and whenever poked, until ctx is done.
func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(runnerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.RunOnce(ctx)
		case <-r.poke:
			r.RunOnce(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// RunOnce embeds all memos currently missing an embedding for the configured
// model, skipping users that are over their daily token cap.
func (r *Runner) RunOnce(ctx context.Context) {
	aiSetting, err := r.store.GetInstanceAISetting(ctx)
	if err != nil {
		slog.Error("embedding runner: failed to get AI setting", "error", err)
		return
	}
	provider, model, ok := ResolveProvider(aiSetting)
	if !ok {
		// Semantic search is not configured; nothing to do.
		return
	}
	embedder, err := r.newEmbedder(provider)
	if err != nil {
		slog.Error("embedding runner: failed to build embedder", "error", err)
		return
	}

	for {
		memos, err := r.store.ListMemosNeedingEmbedding(ctx, model, batchSize)
		if err != nil {
			slog.Error("embedding runner: failed to list memos", "error", err)
			return
		}
		if len(memos) == 0 {
			return
		}

		embedded, err := r.embedBatch(ctx, embedder, model, memos)
		if err != nil {
			// Provider or store failure: leave the rest for the next tick.
			slog.Error("embedding runner: batch failed", "error", err)
			return
		}
		if embedded == 0 {
			// Every memo in the batch belongs to capped users; stop so the
			// missing-row scan does not spin until the next UTC day.
			return
		}
	}
}

func (r *Runner) embedBatch(ctx context.Context, embedder aiembedding.Embedder, model string, memos []*store.Memo) (int, error) {
	byCreator := map[int32][]*store.Memo{}
	for _, memo := range memos {
		byCreator[memo.CreatorID] = append(byCreator[memo.CreatorID], memo)
	}

	usageDate := plan.UsageDate(time.Now())
	embedded := 0
	for creatorID, creatorMemos := range byCreator {
		limits := plan.ForUser(r.profile, nil)
		if limits.AITokensPerDay > 0 {
			used, err := r.store.GetUserAIUsage(ctx, creatorID, usageDate)
			if err != nil {
				return embedded, errors.Wrap(err, "failed to get AI usage")
			}
			if used >= limits.AITokensPerDay {
				continue
			}
		}

		inputs := make([]string, 0, len(creatorMemos))
		for _, memo := range creatorMemos {
			inputs = append(inputs, truncateRunes(memo.Content, maxEmbedRunes))
		}
		resp, err := embedder.Embed(ctx, aiembedding.Request{
			Input:    inputs,
			Model:    model,
			TaskType: aiembedding.TaskDocument,
		})
		if err != nil {
			return embedded, errors.Wrap(err, "failed to embed memos")
		}
		if len(resp.Vectors) != len(creatorMemos) {
			return embedded, errors.Errorf("embedder returned %d vectors for %d memos", len(resp.Vectors), len(creatorMemos))
		}

		for i, memo := range creatorMemos {
			if _, err := r.store.UpsertMemoEmbedding(ctx, &store.MemoEmbedding{
				MemoID:    memo.ID,
				CreatorID: memo.CreatorID,
				Model:     model,
				Vector:    aiembedding.EncodeVector(resp.Vectors[i]),
			}); err != nil {
				return embedded, errors.Wrap(err, "failed to upsert embedding")
			}
			embedded++
		}
		if err := r.store.IncrementUserAIUsage(ctx, creatorID, usageDate, resp.TokensUsed); err != nil {
			return embedded, errors.Wrap(err, "failed to record AI usage")
		}
	}
	return embedded, nil
}

// ResolveProvider returns the configured embedding provider and model, or
// ok=false when semantic search is not configured. Shared with the search API.
func ResolveProvider(setting *storepb.InstanceAISetting) (ai.ProviderConfig, string, bool) {
	providerID := setting.GetEmbedding().GetProviderId()
	if providerID == "" {
		return ai.ProviderConfig{}, "", false
	}
	for _, provider := range setting.GetProviders() {
		if provider == nil || provider.Id != providerID {
			continue
		}
		cfg := ai.ProviderConfig{
			ID:       provider.GetId(),
			Title:    provider.GetTitle(),
			Endpoint: provider.GetEndpoint(),
			APIKey:   provider.GetApiKey(),
		}
		switch provider.GetType() {
		case storepb.AIProviderType_OPENAI:
			cfg.Type = ai.ProviderOpenAI
		case storepb.AIProviderType_GEMINI:
			cfg.Type = ai.ProviderGemini
		default:
			return ai.ProviderConfig{}, "", false
		}
		model := setting.GetEmbedding().GetModel()
		if model == "" {
			defaultModel, err := ai.DefaultEmbeddingModel(cfg.Type)
			if err != nil {
				slog.Error("embedding runner: no default model for provider", "error", err)
				return ai.ProviderConfig{}, "", false
			}
			model = defaultModel
		}
		return cfg, model, true
	}
	return ai.ProviderConfig{}, "", false
}

// NewEmbedder builds the provider-appropriate embedder. Shared with the search API.
func NewEmbedder(cfg ai.ProviderConfig) (aiembedding.Embedder, error) {
	switch cfg.Type {
	case ai.ProviderGemini:
		return gemini.New(cfg, aiembedding.ApplyOptions(nil))
	default:
		return nil, errors.Wrapf(ai.ErrEmbeddingNotSupported, "provider type %q", cfg.Type)
	}
}

func truncateRunes(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit])
}
