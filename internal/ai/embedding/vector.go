package embedding

import (
	"encoding/binary"
	"math"

	"github.com/pkg/errors"
)

// EncodeVector serializes a vector as little-endian float32 bytes for BLOB storage.
func EncodeVector(vector []float32) []byte {
	buf := make([]byte, len(vector)*4)
	for i, v := range vector {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

// DecodeVector deserializes little-endian float32 bytes produced by EncodeVector.
func DecodeVector(data []byte) ([]float32, error) {
	if len(data)%4 != 0 {
		return nil, errors.Errorf("vector blob length %d is not a multiple of 4", len(data))
	}
	vector := make([]float32, len(data)/4)
	for i := range vector {
		vector[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
	}
	return vector, nil
}

// CosineSimilarity returns the cosine similarity of two vectors, or 0 when the
// dimensions differ or either vector has zero magnitude.
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}
