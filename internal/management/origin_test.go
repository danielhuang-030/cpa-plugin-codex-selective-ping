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
