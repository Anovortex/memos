package embedding

import (
	"net/http"
	"time"
)

const defaultHTTPTimeout = 2 * time.Minute

// Options is the resolved option set passed to provider implementations.
type Options struct {
	HTTPClient *http.Client
}

// EmbedderOption customizes an Embedder.
type EmbedderOption func(*Options)

// WithHTTPClient overrides the HTTP client used by the embedder.
func WithHTTPClient(client *http.Client) EmbedderOption {
	return func(o *Options) {
		if client != nil {
			o.HTTPClient = client
		}
	}
}

// ApplyOptions resolves an EmbedderOption slice into Options with defaults.
func ApplyOptions(opts []EmbedderOption) Options {
	resolved := Options{HTTPClient: &http.Client{Timeout: defaultHTTPTimeout}}
	for _, apply := range opts {
		apply(&resolved)
	}
	return resolved
}
