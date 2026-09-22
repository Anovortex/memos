// Package embedding defines the provider-agnostic text-embedding contract used
// by semantic search: the indexer embeds memo content, the search handler embeds
// queries, and providers implement Embedder.
package embedding

import "context"

// Task types hint providers how the embedding will be used; providers without
// task-type support ignore them.
const (
	TaskDocument = "RETRIEVAL_DOCUMENT"
	TaskQuery    = "RETRIEVAL_QUERY"
)

// Embedder produces embedding vectors for a batch of texts.
type Embedder interface {
	Embed(ctx context.Context, req Request) (*Response, error)
}

// Request is a batch embedding request. Inputs must be pre-truncated by the caller.
type Request struct {
	Input    []string
	Model    string
	TaskType string
}

// Response carries one vector per input, in input order, plus the token usage
// to meter against the caller's daily cap (estimated when the provider does not
// report it).
type Response struct {
	Vectors    [][]float32
	TokensUsed int64
}
