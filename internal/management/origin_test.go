package management

import "testing"

func TestOriginFromHeadersHostOnly(t *testing.T) {
	got := originFromHeaders(map[string][]string{"Host": {"cpa.example:8317"}})
	if got != "http://cpa.example:8317" {
		t.Fatalf("got %q", got)
	}
}

func TestOriginFromHeadersForwarded(t *testing.T) {
	got := originFromHeaders(map[string][]string{
		"X-Forwarded-Proto": {"https"},
		"X-Forwarded-Host":  {"cpa.example"},
		"Host":              {"localhost:8317"},
	})
	if got != "https://cpa.example" {
		t.Fatalf("got %q", got)
	}
}

func TestOriginFromHeadersXCspOriginWins(t *testing.T) {
	got := originFromHeaders(map[string][]string{
		"X-Csp-Origin":      {"https://cpa.example:8317"},
		"Host":              {"localhost:9999"},
		"X-Forwarded-Proto": {"http"},
		"X-Forwarded-Host":  {"forwarded.example"},
	})
	if got != "https://cpa.example:8317" {
		t.Fatalf("X-Csp-Origin should win: got %q", got)
	}
}

func TestOriginFromHeadersXCspOriginTrimTrailingSlash(t *testing.T) {
	got := originFromHeaders(map[string][]string{
		"X-Csp-Origin": {"https://cpa.example:8317/"},
	})
	if got != "https://cpa.example:8317" {
		t.Fatalf("trailing slash should be trimmed: got %q", got)
	}
}

func TestOriginFromHeadersEmpty(t *testing.T) {
	if got := originFromHeaders(nil); got != "" {
		t.Fatalf("nil headers: got %q", got)
	}
	if got := originFromHeaders(map[string][]string{}); got != "" {
		t.Fatalf("empty headers: got %q", got)
	}
	if got := originFromHeaders(map[string][]string{"X-Forwarded-Proto": {"https"}}); got != "" {
		t.Fatalf("proto without host: got %q", got)
	}
}
