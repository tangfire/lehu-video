package service

import (
	"net/http"
	"testing"
)

func TestClientIPDoesNotTrustForwardedHeaderFromPublicRemote(t *testing.T) {
	t.Setenv("LEHU_TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
	req := &http.Request{
		RemoteAddr: "203.0.113.10:4567",
		Header: http.Header{
			"X-Forwarded-For": []string{"198.51.100.8"},
		},
	}

	if got := clientIP(req); got != "203.0.113.10" {
		t.Fatalf("clientIP() = %q, want public remote address", got)
	}
}

func TestClientIPTrustsForwardedHeaderFromTrustedProxy(t *testing.T) {
	t.Setenv("LEHU_TRUSTED_PROXY_CIDRS", "172.16.0.0/12")
	req := &http.Request{
		RemoteAddr: "172.18.0.5:4567",
		Header: http.Header{
			"X-Forwarded-For": []string{"198.51.100.8, 172.18.0.1"},
		},
	}

	if got := clientIP(req); got != "198.51.100.8" {
		t.Fatalf("clientIP() = %q, want forwarded client IP", got)
	}
}

func TestClientIPTrustsRealIPFromTrustedProxy(t *testing.T) {
	t.Setenv("LEHU_TRUSTED_PROXY_CIDRS", "127.0.0.0/8")
	req := &http.Request{
		RemoteAddr: "127.0.0.1:4567",
		Header: http.Header{
			"X-Real-Ip": []string{"198.51.100.9"},
		},
	}

	if got := clientIP(req); got != "198.51.100.9" {
		t.Fatalf("clientIP() = %q, want real client IP", got)
	}
}

func TestClientIPHandlesIPv6RemoteAddr(t *testing.T) {
	req := &http.Request{
		RemoteAddr: "[2001:db8::1]:4567",
		Header: http.Header{
			"X-Forwarded-For": []string{"198.51.100.8"},
		},
	}

	if got := clientIP(req); got != "2001:db8::1" {
		t.Fatalf("clientIP() = %q, want IPv6 remote address", got)
	}
}
