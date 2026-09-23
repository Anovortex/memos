package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/usememos/memos/internal/ai/embedding"
	"github.com/usememos/memos/provider/ai"
)

func newTestEmbedder(t *testing.T, handler http.HandlerFunc) *Embedder {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	embedder, err := New(ai.ProviderConfig{
		Type:     ai.ProviderGemini,
		Endpoint: server.URL + "/v1beta",
		APIKey:   "test-key",
	}, embedding.ApplyOptions(nil))
	require.NoError(t, err)
	return embedder
}

func TestGeminiEmbedReturnsVectorsAndEstimatedTokens(t *testing.T) {
	// Arrange
	var gotBody map[string]any
	embedder := newTestEmbedder(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[{"values":[0.1,0.2]},{"values":[0.3,0.4]}]}`))
	})

	// Act
	resp, err := embedder.Embed(context.Background(), embedding.Request{
		Input:    []string{"first memo text", "second memo"},
		Model:    "gemini-embedding-001",
		TaskType: embedding.TaskDocument,
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, [][]float32{{0.1, 0.2}, {0.3, 0.4}}, resp.Vectors)
	// 15 runes -> 4 tokens, 11 runes -> 3 tokens (ceil(runes/4)).
	require.Equal(t, int64(7), resp.TokensUsed)
	require.NotNil(t, gotBody)
}

func TestGeminiEmbedCountMismatchFails(t *testing.T) {
	embedder := newTestEmbedder(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[{"values":[0.1]}]}`))
	})

	_, err := embedder.Embed(context.Background(), embedding.Request{
		Input: []string{"one", "two"},
		Model: "gemini-embedding-001",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "embeddings for 2 inputs")
}

func TestGeminiEmbedAPIErrorPropagates(t *testing.T) {
	embedder := newTestEmbedder(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":{"message":"quota exceeded"}}`, http.StatusTooManyRequests)
	})

	_, err := embedder.Embed(context.Background(), embedding.Request{
		Input: []string{"text"},
		Model: "gemini-embedding-001",
	})
	require.Error(t, err)
}

func TestGeminiEmbedValidation(t *testing.T) {
	embedder := newTestEmbedder(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	_, err := embedder.Embed(context.Background(), embedding.Request{Input: []string{"x"}})
	require.Error(t, err, "missing model should fail")

	_, err = embedder.Embed(context.Background(), embedding.Request{Model: "m"})
	require.Error(t, err, "missing input should fail")
}

func TestGeminiNewRequiresAPIKey(t *testing.T) {
	_, err := New(ai.ProviderConfig{Type: ai.ProviderGemini}, embedding.ApplyOptions(nil))
	require.Error(t, err)
}
