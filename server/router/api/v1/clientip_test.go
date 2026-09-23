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
		{"public peer ignores forwarded headers", ctxWithMetadata(peerMetadataKey, "203.0.113.9:51234", "x-forwarded-for", "198.51.100.1", cfConnectingIPKey, "198.51.100.2"), "203.0.113.9"},
		{"loopback peer prefers cloudflare header", ctxWithMetadata(peerMetadataKey, "127.0.0.1:40000", cfConnectingIPKey, "198.51.100.9", "x-forwarded-for", "203.0.113.66"), "198.51.100.9"},
		{"loopback peer takes right-most untrusted forwarded entry", ctxWithMetadata(peerMetadataKey, "127.0.0.1:40000", "x-forwarded-for", " 198.51.100.1 , 10.0.0.2"), "198.51.100.1"},
		{"visitor-supplied left entry is ignored", ctxWithMetadata(peerMetadataKey, "127.0.0.1:40000", "x-forwarded-for", "203.0.113.66, 198.51.100.1, 127.0.0.1"), "198.51.100.1"},
		{"private peer trusts forwarded entry", ctxWithMetadata(peerMetadataKey, "10.0.0.5:40000", "x-forwarded-for", "198.51.100.7"), "198.51.100.7"},
		{"all-private chain falls back to left-most", ctxWithMetadata(peerMetadataKey, "127.0.0.1:40000", "x-forwarded-for", "10.0.0.1, 10.0.0.2"), "10.0.0.1"},
		{"ipv6 loopback peer", ctxWithMetadata(peerMetadataKey, "[::1]:40000", "x-forwarded-for", "2001:db8::1"), "2001:db8::1"},
		{"loopback peer without headers", ctxWithMetadata(peerMetadataKey, "127.0.0.1:40000"), "127.0.0.1"},
		{"peer without port", ctxWithMetadata(peerMetadataKey, "203.0.113.9"), "203.0.113.9"},
		{"forwarded header only", ctxWithMetadata("x-forwarded-for", "198.51.100.3, 127.0.0.1"), "198.51.100.3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, clientIP(tt.ctx))
		})
	}
}
