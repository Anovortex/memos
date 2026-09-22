package ai

import "github.com/pkg/errors"

const (
	// DefaultOpenAITranscriptionModel is the built-in OpenAI transcription model.
	DefaultOpenAITranscriptionModel = "whisper-1"
	// DefaultGeminiTranscriptionModel is the built-in Gemini transcription model.
	DefaultGeminiTranscriptionModel = "gemini-2.5-flash"
	// DefaultGeminiEmbeddingModel is the built-in Gemini embedding model.
	DefaultGeminiEmbeddingModel = "gemini-embedding-001"
)

// DefaultTranscriptionModel returns the built-in transcription model for a provider.
func DefaultTranscriptionModel(providerType ProviderType) (string, error) {
	switch providerType {
	case ProviderOpenAI:
		return DefaultOpenAITranscriptionModel, nil
	case ProviderGemini:
		return DefaultGeminiTranscriptionModel, nil
	default:
		return "", errors.Wrapf(ErrCapabilityUnsupported, "provider type %q", providerType)
	}
}

// DefaultEmbeddingModel returns the built-in embedding model for a provider.
// Only Gemini is supported for now; an OpenAI-compatible embedder is a planned
// follow-up.
func DefaultEmbeddingModel(providerType ProviderType) (string, error) {
	switch providerType {
	case ProviderGemini:
		return DefaultGeminiEmbeddingModel, nil
	default:
		return "", errors.Wrapf(ErrEmbeddingNotSupported, "provider type %q", providerType)
	}
}
