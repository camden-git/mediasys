package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

var testTrustedProxies = []netip.Prefix{
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("fc00::/7"),
}

func TestResolveClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        []string
		want       string
		wantOK     bool
	}{
		{"direct public peer", "203.0.113.7:5555", nil, "", false},
		{"direct public peer with spoofed XFF", "203.0.113.7:5555", []string{"198.51.100.1"}, "", false},
		{"trusted peer without XFF", "172.18.0.3:4000", nil, "172.18.0.3", true},
		{"nginx only", "172.18.0.3:4000", []string{"203.0.113.7"}, "203.0.113.7", true},
		{"outer proxy and nginx", "172.18.0.3:4000", []string{"203.0.113.7, 172.18.0.1"}, "203.0.113.7", true},
		{"client-supplied entries on the left", "172.18.0.3:4000", []string{"1.1.1.1, 2.2.2.2, 203.0.113.7, 172.18.0.1"}, "203.0.113.7", true},
		{"spoofed trusted entry left of client", "172.18.0.3:4000", []string{"10.0.0.1, 203.0.113.7"}, "203.0.113.7", true},
		{"multiple header lines", "172.18.0.3:4000", []string{"1.1.1.1", "203.0.113.7, 127.0.0.1"}, "203.0.113.7", true},
		{"all trusted", "127.0.0.1:4000", []string{"10.0.0.5, 192.168.1.2"}, "10.0.0.5", true},
		{"malformed rightmost", "172.18.0.3:4000", []string{"203.0.113.7, garbage"}, "172.18.0.3", true},
		{"malformed left of trusted hop", "172.18.0.3:4000", []string{"garbage, 10.0.0.1"}, "10.0.0.1", true},
		{"empty entry", "172.18.0.3:4000", []string{"203.0.113.7, "}, "172.18.0.3", true},
		{"hop with port", "172.18.0.3:4000", []string{"203.0.113.7:1234"}, "203.0.113.7", true},
		{"malformed remote addr", "not-an-ip", []string{"203.0.113.7"}, "", false},
		{"ipv6 client", "[::1]:4000", []string{"2001:db8::1"}, "2001:db8::1", true},
		{"ipv6 client with port", "[fd00::2]:4000", []string{"[2001:db8::1]:443, fd00::1"}, "2001:db8::1", true},
		{"ipv6 public peer", "[2001:db8::9]:4000", []string{"198.51.100.1"}, "", false},
		{"ipv4-mapped peer", "[::ffff:172.18.0.3]:4000", []string{"::ffff:203.0.113.7"}, "203.0.113.7", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := resolveClientIP(tt.remoteAddr, tt.xff, testTrustedProxies)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("resolveClientIP(%q, %q) = %q, %v; want %q, %v", tt.remoteAddr, tt.xff, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestClientIPMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{"public peer ignores headers", "203.0.113.7:5555", map[string]string{
			"X-Forwarded-For": "198.51.100.1", "True-Client-IP": "198.51.100.2", "X-Real-IP": "198.51.100.3",
		}, "203.0.113.7"},
		{"trusted peer ignores True-Client-IP and X-Real-IP", "172.18.0.3:4000", map[string]string{
			"X-Forwarded-For": "203.0.113.7", "True-Client-IP": "198.51.100.2", "X-Real-IP": "198.51.100.3",
		}, "203.0.113.7"},
		{"trusted peer without XFF", "127.0.0.1:4000", map[string]string{"True-Client-IP": "198.51.100.2"}, "127.0.0.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			h := clientIP(testTrustedProxies)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = getClientIP(r)
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			if got != tt.want {
				t.Fatalf("client IP = %q, want %q", got, tt.want)
			}
		})
	}
}
