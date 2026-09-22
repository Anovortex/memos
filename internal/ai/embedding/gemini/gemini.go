// Package gemini implements embedding.Embedder against the Gemini embedContent
// endpoint. Used by semantic search when the configured embedding provider is a
// Gemini provider.
package gemini

import (
	"context"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"
	"google.golang.org/genai"

	"github.com/usememos/memos/internal/ai"
	"github.com/usememos/memos/internal/ai/embedding"
)

const (
	defaultEndpoint   = "https://generativelanguage.googleapis.com/v1beta"
	defaultAPIVersion = "v1beta"
	providerName      = "Gemini"

	// estimatedRunesPerToken approximates token usage because the Gemini API
	// (unlike Vertex) reports no token counts for embedContent.
	estimatedRunesPerToken = 4
)

// Embedder implements embedding.Embedder for Gemini embedContent.
type Embedder struct {
	client *genai.Client
}

// New constructs an Embedder from a provider config.
func New(cfg ai.ProviderConfig, options embedding.Options) (*Embedder, error) {
	endpoint, err := normalizeEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	if cfg.APIKey == "" {
		return nil, errors.Errorf("%s API key is required", providerName)
	}
	baseURL, apiVersion, err := splitEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	httpOptions := genai.HTTPOptions{BaseURL: baseURL, APIVersion: apiVersion}
	if options.HTTPClient != nil && options.HTTPClient.Timeout > 0 {
		timeout := options.HTTPClient.Timeout
		httpOptions.Timeout = &timeout
	}
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:      cfg.APIKey,
		Backend:     genai.BackendGeminiAPI,
		HTTPClient:  options.HTTPClient,
		HTTPOptions: httpOptions,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Gemini client")
	}
	return &Embedder{client: client}, nil
}

// Embed calls Gemini embedContent for the batch of inputs.
func (e *Embedder) Embed(ctx context.Context, req embedding.Request) (*embedding.Response, error) {
	if strings.TrimSpace(req.Model) == "" {
		return nil, errors.New("model is required")
	}
	if len(req.Input) == 0 {
		return nil, errors.New("input is required")
	}

	contents := make([]*genai.Content, 0, len(req.Input))
	for _, text := range req.Input {
		contents = append(contents, genai.NewContentFromText(text, genai.RoleUser))
	}
	cfg := &genai.EmbedContentConfig{TaskType: req.TaskType}

	resp, err := e.client.Models.EmbedContent(ctx, normalizeModelName(req.Model), contents, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send Gemini embedding request")
	}
	if len(resp.Embeddings) != len(req.Input) {
		return nil, errors.Errorf("Gemini returned %d embeddings for %d inputs", len(resp.Embeddings), len(req.Input))
	}

	vectors := make([][]float32, 0, len(resp.Embeddings))
	for _, emb := range resp.Embeddings {
		if emb == nil || len(emb.Values) == 0 {
			return nil, errors.New("Gemini returned an empty embedding")
		}
		vectors = append(vectors, emb.Values)
	}

	return &embedding.Response{
		Vectors:    vectors,
		TokensUsed: estimateTokens(req.Input),
	}, nil
}

func estimateTokens(inputs []string) int64 {
	var tokens int64
	for _, text := range inputs {
		runes := utf8.RuneCountInString(text)
		tokens += int64((runes + estimatedRunesPerToken - 1) / estimatedRunesPerToken)
	}
	return tokens
}

func normalizeEndpoint(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	if _, err := url.ParseRequestURI(endpoint); err != nil {
		return "", errors.Wrapf(err, "invalid %s endpoint", providerName)
	}
	return strings.TrimRight(endpoint, "/"), nil
}

func splitEndpoint(endpoint string) (string, string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", "", errors.Wrap(err, "invalid Gemini endpoint")
	}
	path := strings.TrimRight(parsed.Path, "/")
	apiVersion := defaultAPIVersion
	for _, supported := range []string{"v1alpha", "v1beta", "v1"} {
		if path == "/"+supported || strings.HasSuffix(path, "/"+supported) {
			apiVersion = supported
			parsed.Path = strings.TrimSuffix(path, "/"+supported)
			break
		}
	}
	return strings.TrimRight(parsed.String(), "/"), apiVersion, nil
}

func normalizeModelName(model string) string {
	return strings.TrimPrefix(strings.TrimSpace(model), "models/")
}
