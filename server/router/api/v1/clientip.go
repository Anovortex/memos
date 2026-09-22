package v1

import (
	"context"
	"net"
	"strings"

	"google.golang.org/grpc/metadata"
)

const (
	// peerMetadataKey carries the transport-level remote address. The Connect
	// interceptor and the gateway mux both set it from the socket, so it cannot
	// be forged by a client header.
	peerMetadataKey = "x-memos-peer"
	unknownClientIP = "unknown"
)

// clientIP resolves the caller's address for rate limiting.
//
// The peer address is trusted as-is when it is public. When it is loopback or
// private, the request reached us through a proxy we run (cloudflared on the
// same host, or a LAN reverse proxy), so the first X-Forwarded-For entry is the
// visitor. A public client that sends its own X-Forwarded-For is ignored.
func clientIP(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return unknownClientIP
	}
	forwarded := firstForwardedFor(md)
	peer := hostOnly(firstMetadataValue(md, peerMetadataKey))
	if peer == "" {
		if forwarded != "" {
			return forwarded
		}
		return unknownClientIP
	}
	if forwarded != "" && isTrustedProxyAddr(peer) {
		return forwarded
	}
	return peer
}

func firstMetadataValue(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func firstForwardedFor(md metadata.MD) string {
	raw := firstMetadataValue(md, "x-forwarded-for")
	if raw == "" {
		return ""
	}
	first, _, _ := strings.Cut(raw, ",")
	return strings.TrimSpace(first)
}

// hostOnly strips a port from "host:port" or "[v6]:port"; bare hosts pass through.
func hostOnly(addr string) string {
	if addr == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return strings.Trim(addr, "[]")
}

// isTrustedProxyAddr reports whether addr is loopback, private, or link-local,
// i.e. a proxy on our side of the network rather than the visitor.
func isTrustedProxyAddr(addr string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}
