package management

import (
	"context"
	"encoding/json"
	"testing"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/pinger"
)

type modelsHost struct {
	files   []hostapi.AuthFile
	creds   map[string][]byte
	calls   []hostapi.HTTPRequest
	script  []hostapi.HTTPResponse
	httpErr error
}

func (m *modelsHost) AuthList(context.Context) ([]hostapi.AuthFile, error) { return m.files, nil }
func (m *modelsHost) AuthGet(_ context.Context, idx string) ([]byte, error) {
	if m.creds != nil {
		if raw, ok := m.creds[idx]; ok {
			return raw, nil
		}
	}
	return json.Marshal(map[string]any{"access_token": "codex-tok"})
}
func (m *modelsHost) AuthGetRuntime(context.Context, string) (json.RawMessage, error) {
	return nil, hostapi.ErrUnsupported
}
func (m *modelsHost) HTTPDo(_ context.Context, req hostapi.HTTPRequest) (hostapi.HTTPResponse, error) {
	m.calls = append(m.calls, req)
	if m.httpErr != nil {
		return hostapi.HTTPResponse{}, m.httpErr
	}
	i := len(m.calls) - 1
	if i >= len(m.script) {
		i = len(m.script) - 1
	}
	if i < 0 {
		return hostapi.HTTPResponse{StatusCode: 500, Body: []byte(`{}`)}, nil
	}
	return m.script[i], nil
}

func TestBearerFromAuthHeader(t *testing.T) {
	got := bearerFromAuthHeader(map[string][]string{"Authorization": {"Bearer mgmt-secret"}})
	if got != "mgmt-secret" {
		t.Fatalf("got %q", got)
	}
	if bearerFromAuthHeader(nil) != "" {
		t.Fatal("empty headers")
	}
}





func TestFetchFilteredModelsAPIKeyOK(t *testing.T) {
	keysBody, _ := json.Marshal(map[string]any{"api-keys": []string{"sk-proxy"}})
	modelsBody, _ := json.Marshal(map[string]any{
		"data": []map[string]any{
			{"id": "gpt-6-luna", "owned_by": "openai"},
			{"id": "claude-3", "owned_by": "anthropic"},
		},
	})
	h := &modelsHost{script: []hostapi.HTTPResponse{
		{StatusCode: 200, Body: keysBody},
		{StatusCode: 200, Body: modelsBody},
	}}
	out := fetchFilteredModels(context.Background(), h, "http://cpa.test", "mgmt-key", "")
	if out.Warning != "" {
		t.Fatalf("warning=%q", out.Warning)
	}
	if len(h.calls) != 2 {
		t.Fatalf("calls=%d", len(h.calls))
	}
	if h.calls[0].URL != "http://cpa.test/v0/management/api-keys" {
		t.Fatalf("keys url=%q", h.calls[0].URL)
	}
	if firstHeader(h.calls[0].Headers, "Authorization") != "Bearer mgmt-key" {
		t.Fatalf("keys auth=%q", firstHeader(h.calls[0].Headers, "Authorization"))
	}
	if h.calls[1].URL != "http://cpa.test/v1/models" {
		t.Fatalf("models url=%q", h.calls[1].URL)
	}
	if firstHeader(h.calls[1].Headers, "Authorization") != "Bearer sk-proxy" {
		t.Fatalf("models auth=%q", firstHeader(h.calls[1].Headers, "Authorization"))
	}
	ids := modelIDs(out)
	if !hasID(ids, "gpt-6-luna") || hasID(ids, "claude-3") {
		t.Fatalf("ids=%v", ids)
	}
}

func TestFetchFilteredModelsAPIKeysFailThenCodexOK(t *testing.T) {
	okBody, _ := json.Marshal(map[string]any{
		"data": []map[string]any{{"id": "gpt-4o", "owned_by": "openai"}},
	})
	h := &modelsHost{
		files: []hostapi.AuthFile{{AuthIndex: "1", Provider: "codex", Name: "a"}},
		creds: map[string][]byte{"1": mustJSON(map[string]any{"access_token": "codex-tok"})},
		script: []hostapi.HTTPResponse{
			{StatusCode: 401, Body: []byte(`{"error":"no"}`)},
			{StatusCode: 200, Body: okBody},
		},
	}
	out := fetchFilteredModels(context.Background(), h, "https://cpa.test", "mgmt-key", "kept-orphan")
	if out.Warning != "" {
		t.Fatalf("warning=%q", out.Warning)
	}
	if len(h.calls) != 2 {
		t.Fatalf("calls=%d", len(h.calls))
	}
	if h.calls[0].URL != "https://cpa.test/v0/management/api-keys" {
		t.Fatalf("first url=%q", h.calls[0].URL)
	}
	if firstHeader(h.calls[1].Headers, "Authorization") != "Bearer codex-tok" {
		t.Fatalf("second auth=%q", firstHeader(h.calls[1].Headers, "Authorization"))
	}
	if !hasID(modelIDs(out), "gpt-4o") {
		t.Fatalf("ids=%v", modelIDs(out))
	}
}

func TestFetchFilteredModelsBothFailFallback(t *testing.T) {
	h := &modelsHost{
		files: []hostapi.AuthFile{{AuthIndex: "1", Provider: "codex"}},
		script: []hostapi.HTTPResponse{
			{StatusCode: 401, Body: []byte(`{}`)},
			{StatusCode: 401, Body: []byte(`{}`)},
		},
	}
	out := fetchFilteredModels(context.Background(), h, "http://cpa.test", "mgmt-key", "gpt-5.6-luna")
	if out.Warning == "" {
		t.Fatal("expected warning")
	}
	ids := modelIDs(out)
	if !hasID(ids, "gpt-5.6-luna") || !hasID(ids, pinger.ModelName) {
		t.Fatalf("ids=%v want configured + %s", ids, pinger.ModelName)
	}
}

func modelIDs(out modelsListResponse) []string {
	ids := make([]string, 0, len(out.Data))
	for _, d := range out.Data {
		ids = append(ids, d.ID)
	}
	return ids
}

func hasID(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func TestFirstAPIKeyFromBodyStringArray(t *testing.T) {
	got, err := firstAPIKeyFromBody([]byte(`{"api-keys":[" sk-a ","sk-b"]}`))
	if err != nil || got != "sk-a" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestFirstAPIKeyFromBodyObjectArray(t *testing.T) {
	got, err := firstAPIKeyFromBody([]byte(`{"api-keys":[{"api-key":"sk-obj"},{"key":"sk-2"}]}`))
	if err != nil || got != "sk-obj" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestFirstAPIKeyFromBodyEmpty(t *testing.T) {
	if _, err := firstAPIKeyFromBody([]byte(`{"api-keys":[]}`)); err == nil {
		t.Fatal("expected error")
	}
	if _, err := firstAPIKeyFromBody([]byte(`{}`)); err == nil {
		t.Fatal("expected error")
	}
}

