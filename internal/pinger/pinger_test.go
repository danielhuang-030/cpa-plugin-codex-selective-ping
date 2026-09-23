package pinger

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
)

type mockHost struct {
	token     string
	account   string
	status    int
	body      []byte
	calls     int
	lastModel string
}

func (m *mockHost) AuthList(context.Context) ([]hostapi.AuthFile, error) {
	return nil, errors.New("unused")
}
func (m *mockHost) AuthGet(context.Context, string) ([]byte, error) {
	raw, _ := json.Marshal(map[string]any{"access_token": m.token, "account_id": m.account})
	return raw, nil
}
func (m *mockHost) HTTPDo(ctx context.Context, req hostapi.HTTPRequest) (hostapi.HTTPResponse, error) {
	m.calls++
	if req.Method != "POST" || !strings.Contains(req.URL, "codex/responses") {
		tpanic("bad request")
	}
	var body map[string]any
	_ = json.Unmarshal(req.Body, &body)
	if s, ok := body["model"].(string); ok {
		m.lastModel = s
	}
	return hostapi.HTTPResponse{StatusCode: m.status, Body: m.body}, nil
}
func (m *mockHost) AuthGetRuntime(context.Context, string) (json.RawMessage, error) {
	return nil, hostapi.ErrUnsupported
}

func tpanic(s string) { panic(s) }

func TestPingSuccess(t *testing.T) {
	h := &mockHost{token: "tok", account: "acc", status: 200}
	out := PingAccount(context.Background(), h, hostapi.AuthFile{AuthIndex: "1", Name: "a"}, true, time.Time{}, ModelName)
	if out.Status != "success" || out.HTTPStatus != 200 || out.Attempts != 1 {
		t.Fatalf("%#v", out)
	}
}

func TestPingLimitedUsage(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"error": map[string]any{"type": "usage_limit_reached", "message": "slow down", "resets_at": time.Now().Add(time.Hour).Unix()},
	})
	h := &mockHost{token: "tok", status: 429, body: body}
	out := PingAccount(context.Background(), h, hostapi.AuthFile{AuthIndex: "1"}, true, time.Time{}, ModelName)
	if out.Status != "limited" {
		t.Fatalf("%#v", out)
	}
	if out.ResetsAt.IsZero() {
		t.Fatal("expected resets_at")
	}
}

func TestPingRetriesThenFails(t *testing.T) {
	h := &mockHost{token: "tok", status: 500, body: []byte(`{"error":{"message":"boom"}}`)}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// Shrink delays for test by using a test hook if present; otherwise accept real backoff with short MaxAttempts via build tag.
	// Implementation MUST expose `var retryDelayFn = retryDelay` for tests:
	old := retryDelayFn
	retryDelayFn = func(int) time.Duration { return time.Millisecond }
	defer func() { retryDelayFn = old }()
	out := PingAccount(ctx, h, hostapi.AuthFile{AuthIndex: "1"}, true, time.Time{}, ModelName)
	if out.Status != "failed" || out.Attempts != MaxAttempts {
		t.Fatalf("%#v calls=%d", out, h.calls)
	}
	if h.calls != MaxAttempts {
		t.Fatalf("calls=%d", h.calls)
	}
}

func TestPingBare429LimitedNoRetry(t *testing.T) {
	h := &mockHost{token: "tok", status: 429, body: []byte(`{}`)}
	old := retryDelayFn
	retryDelayFn = func(int) time.Duration { return time.Millisecond }
	defer func() { retryDelayFn = old }()
	out := PingAccount(context.Background(), h, hostapi.AuthFile{AuthIndex: "1"}, true, time.Time{}, ModelName)
	if out.Status != "limited" {
		t.Fatalf("status=%q want limited; %#v", out.Status, out)
	}
	if out.Attempts != 1 {
		t.Fatalf("attempts=%d want 1 (no retry storm)", out.Attempts)
	}
	if h.calls != 1 {
		t.Fatalf("http calls=%d want 1", h.calls)
	}
}

func TestPingUsesInjectedModel(t *testing.T) {
	const want = "gpt-custom-inject"
	h := &mockHost{token: "tok", account: "acc", status: 200}
	out := PingAccount(context.Background(), h, hostapi.AuthFile{AuthIndex: "1"}, true, time.Time{}, want)
	if out.Status != "success" {
		t.Fatalf("status=%q %#v", out.Status, out)
	}
	if h.lastModel != want {
		t.Fatalf("request model=%q want %q", h.lastModel, want)
	}
}

func TestPingDefaultModelNameWhenPassed(t *testing.T) {
	h := &mockHost{token: "tok", status: 200}
	out := PingAccount(context.Background(), h, hostapi.AuthFile{AuthIndex: "1"}, true, time.Time{}, ModelName)
	if out.Status != "success" {
		t.Fatalf("%#v", out)
	}
	if h.lastModel != ModelName {
		t.Fatalf("request model=%q want %q", h.lastModel, ModelName)
	}
}

func TestModelNameIsGPT6Luna(t *testing.T) {
	if ModelName != "gpt-6-luna" {
		t.Fatalf("ModelName=%q want gpt-6-luna", ModelName)
	}
}
