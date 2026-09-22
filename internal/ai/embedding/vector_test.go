package embedding

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVectorEncodeDecodeRoundtrip(t *testing.T) {
	// Arrange
	original := []float32{0.5, -1.25, 3.75, 0}

	// Act
	encoded := EncodeVector(original)
	decoded, err := DecodeVector(encoded)

	// Assert
	require.NoError(t, err)
	require.Equal(t, original, decoded)
	require.Len(t, encoded, len(original)*4)
}

func TestVectorDecodeEmpty(t *testing.T) {
	decoded, err := DecodeVector([]byte{})
	require.NoError(t, err)
	require.Len(t, decoded, 0)
}

func TestVectorDecodeOddLengthFails(t *testing.T) {
	_, err := DecodeVector([]byte{1, 2, 3})
	require.Error(t, err)
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b []float32
		want float32
	}{
		{"identical", []float32{1, 2, 3}, []float32{1, 2, 3}, 1},
		{"orthogonal", []float32{1, 0}, []float32{0, 1}, 0},
		{"opposite", []float32{1, 0}, []float32{-1, 0}, -1},
		{"dimension mismatch", []float32{1, 2}, []float32{1, 2, 3}, 0},
		{"zero vector", []float32{0, 0}, []float32{1, 2}, 0},
		{"empty", []float32{}, []float32{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.InDelta(t, tt.want, CosineSimilarity(tt.a, tt.b), 1e-6)
		})
	}
}
