package runner

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"

	"cpa-plugin-codex-selective-ping/internal/config"
	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/runstate"
)

type mockHost struct {
	files []hostapi.AuthFile
	http  int32
}

func (m *mockHost) AuthList(context.Context) ([]hostapi.AuthFile, error) { return m.files, nil }
func (m *mockHost) AuthGet(context.Context, string) ([]byte, error) {
	return json.Marshal(map[string]any{"access_token": "t", "account_id": "a"})
}
func (m *mockHost) HTTPDo(context.Context, hostapi.HTTPRequest) (hostapi.HTTPResponse, error) {
	atomic.AddInt32(&m.http, 1)
	return hostapi.HTTPResponse{StatusCode: 200}, nil
}
func (m *mockHost) AuthGetRuntime(context.Context, string) (json.RawMessage, error) {
	return nil, hostapi.ErrUnsupported
}

func TestRunEmptyAccountsNoHTTP(t *testing.T) {
	h := &mockHost{files: []hostapi.AuthFile{
		{AuthIndex: "1", Name: "a", Email: "a@x.com", Provider: "codex"},
	}}
	r := &Runner{Host: h, State: runstate.New()}
	sum, ok := r.Run(context.Background(), config.Config{Enabled: true, Timezone: "UTC", Times: []string{"06:00"}, Accounts: nil}, true)
	if !ok {
		t.Fatal("should accept run")
	}
	if sum.Attempted != 0 || atomic.LoadInt32(&h.http) != 0 {
		t.Fatalf("sum=%#v http=%d", sum, h.http)
	}
	if sum.Message == "" {
		t.Fatal("expected message about empty selection")
	}
}

func TestRunOnlySelected(t *testing.T) {
	h := &mockHost{files: []hostapi.AuthFile{
		{AuthIndex: "1", Name: "a", Email: "a@x.com", Provider: "codex"},
		{AuthIndex: "2", Name: "b", Email: "b@x.com", Provider: "codex"},
		{AuthIndex: "3", Name: "g", Email: "g@x.com", Provider: "gemini"},
	}}
	r := &Runner{Host: h, State: runstate.New()}
	sum, ok := r.Run(context.Background(), config.Config{Enabled: true, Timezone: "UTC", Times: []string{"06:00"}, Accounts: []string{"b@x.com"}}, true)
	if !ok {
		t.Fatal("accept")
	}
	if atomic.LoadInt32(&h.http) != 1 {
		t.Fatalf("http calls=%d want 1", h.http)
	}
	if sum.Succeeded != 1 || sum.Attempted != 1 {
		t.Fatalf("%#v", sum)
	}
}

func TestRunMutexRejectsSecond(t *testing.T) {
	h := &mockHost{files: []hostapi.AuthFile{
		{AuthIndex: "1", Name: "a", Email: "a@x.com", Provider: "codex"},
	}}
	st := runstate.New()
	if !st.TryBegin() {
		t.Fatal("prime running")
	}
	r := &Runner{Host: h, State: st}
	_, ok := r.Run(context.Background(), config.Config{Enabled: true, Timezone: "UTC", Times: []string{"06:00"}, Accounts: []string{"a@x.com"}}, true)
	if ok {
		t.Fatal("expected reject while running")
	}
}
