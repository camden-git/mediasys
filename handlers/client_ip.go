package handlers

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// clientIP rewrites r.RemoteAddr to the resolved client IP (see resolveClientIP)
// so rate limiting and request logging see the real client behind trusted proxies.
func clientIP(trusted []netip.Prefix) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ip, ok := resolveClientIP(r.RemoteAddr, r.Header.Values("X-Forwarded-For"), trusted); ok {
				r.RemoteAddr = ip
			}
			next.ServeHTTP(w, r)
		})
	}
}

// resolveClientIP derives the client IP from X-Forwarded-For, but only when the
// immediate peer is a trusted proxy. The chain is walked from the right, since
// each trusted proxy appends the address it received the request from; the first
// untrusted hop is the client and anything left of it is client-supplied. If
// every hop is trusted (or the next one is malformed) the last trusted hop wins.
// Other client IP headers (True-Client-IP, X-Real-IP, ...) are never consulted.
// It reports false when the peer is untrusted and RemoteAddr should stay as is.
func resolveClientIP(remoteAddr string, xff []string, trusted []netip.Prefix) (string, bool) {
	peer, ok := parseHop(remoteAddr)
	if !ok || !isTrusted(peer, trusted) {
		return "", false
	}
	ip := peer
	for i := len(xff) - 1; i >= 0; i-- {
		hops := strings.Split(xff[i], ",")
		for j := len(hops) - 1; j >= 0; j-- {
			hop, ok := parseHop(hops[j])
			if !ok {
				return ip.String(), true
			}
			ip = hop
			if !isTrusted(hop, trusted) {
				return ip.String(), true
			}
		}
	}
	return ip.String(), true
}

// parseHop parses an IP, optionally with a port ("1.2.3.4:80", "[::1]:80").
func parseHop(s string) (netip.Addr, bool) {
	s = strings.TrimSpace(s)
	if host, _, err := net.SplitHostPort(s); err == nil {
		s = host
	}
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Addr{}, false
	}
	return addr.Unmap().WithZone(""), true
}

func isTrusted(ip netip.Addr, trusted []netip.Prefix) bool {
	for _, p := range trusted {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}
