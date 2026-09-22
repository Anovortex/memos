package v1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func ctxWithMetadata(pairs ...string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(pairs...))
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{"no metadata", context.Background(), unknownClientIP},
		{"empty metadata", ctxWithMetadata(), unknownClientIP},
		{"public peer is used as-is", ctxWithMetadata(peerMetadataKey, "203.0.113.9:51234"), "203.0.113.9"},
		{"public peer ignores forwarded header", ctxWithMetadata(peerMetadataKey, "203.0.113.9:51234", "x-forwarded-for", "198.51.100.1"), "203.0.113.9"},
		{"loopback peer trusts first forwarded entry", ctxWithMetadata(peerMetadataKey, "127.0.0.1:40000", "x-forwarded-for", " 198.51.100.1 , 10.0.0.2"), "198.51.100.1"},
		{"private peer trusts forwarded entry", ctxWithMetadata(peerMetadataKey, "10.0.0.5:40000", "x-forwarded-for", "198.51.100.7"), "198.51.100.7"},
		{"ipv6 loopback peer", ctxWithMetadata(peerMetadataKey, "[::1]:40000", "x-forwarded-for", "2001:db8::1"), "2001:db8::1"},
		{"loopback peer without forwarded header", ctxWithMetadata(peerMetadataKey, "127.0.0.1:40000"), "127.0.0.1"},
		{"peer without port", ctxWithMetadata(peerMetadataKey, "203.0.113.9"), "203.0.113.9"},
		{"forwarded header only", ctxWithMetadata("x-forwarded-for", "198.51.100.3, 127.0.0.1"), "198.51.100.3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, clientIP(tt.ctx))
		})
	}
}
