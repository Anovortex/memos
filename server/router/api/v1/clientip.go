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
	// cfConnectingIPKey carries Cloudflare's Cf-Connecting-Ip header. Cloudflare
	// overwrites it on every request through the tunnel, so it is preferred over
	// X-Forwarded-For when the request came from our proxy.
	cfConnectingIPKey = "cf-connecting-ip"
	unknownClientIP   = "unknown"
)

// clientIP resolves the caller's address for rate limiting.
//
// A public socket peer is the client itself and no header is trusted. A
// loopback or private peer means the request came through a proxy we run
// (cloudflared on the same host, a LAN reverse proxy), so the proxy's headers
// are consulted: Cf-Connecting-Ip first, then the right-most X-Forwarded-For
// entry that is not itself one of our proxies. Proxies append to
// X-Forwarded-For, so a value the visitor supplied stays on the left and is
// ignored. A LAN client talking to the server directly can still forge these
// headers; the LAN is inside the trust boundary already.
func clientIP(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return unknownClientIP
	}
	peer := hostOnly(firstMetadataValue(md, peerMetadataKey))
	if peer != "" && !isTrustedProxyAddr(peer) {
		return peer
	}
	if cf := hostOnly(firstMetadataValue(md, cfConnectingIPKey)); cf != "" {
		return cf
	}
	if forwarded := rightmostUntrustedForwardedFor(md); forwarded != "" {
		return forwarded
	}
	if peer != "" {
		return peer
	}
	return unknownClientIP
}

func firstMetadataValue(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

// rightmostUntrustedForwardedFor walks X-Forwarded-For from the right and
// returns the first address that is not one of our proxies. When every entry
// is a proxy address, the left-most is returned as a best effort.
func rightmostUntrustedForwardedFor(md metadata.MD) string {
	entries := []string{}
	for _, raw := range md.Get("x-forwarded-for") {
		for part := range strings.SplitSeq(raw, ",") {
			if entry := hostOnly(strings.TrimSpace(part)); entry != "" {
				entries = append(entries, entry)
			}
		}
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if !isTrustedProxyAddr(entries[i]) {
			return entries[i]
		}
	}
	if len(entries) > 0 {
		return entries[0]
	}
	return ""
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
