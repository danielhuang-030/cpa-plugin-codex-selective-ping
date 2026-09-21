package management

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"cpa-plugin-codex-selective-ping/internal/config"
	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/plugin"
)

type mh struct{ files []hostapi.AuthFile }

func (m *mh) AuthList(context.Context) ([]hostapi.AuthFile, error) { return m.files, nil }
func (m *mh) AuthGet(context.Context, string) ([]byte, error) {
	return json.Marshal(map[string]any{"access_token": "t"})
}
func (m *mh) HTTPDo(context.Context, hostapi.HTTPRequest) (hostapi.HTTPResponse, error) {
	return hostapi.HTTPResponse{StatusCode: 200}, nil
}
func (m *mh) AuthGetRuntime(context.Context, string) (json.RawMessage, error) {
	return nil, hostapi.ErrUnsupported
}

func TestStatusShapeSelected(t *testing.T) {
	p := plugin.New(&mh{files: []hostapi.AuthFile{
		{AuthIndex: "1", Name: "a", Email: "a@x.com", Provider: "codex", Plan: "Plus"},
		{AuthIndex: "2", Name: "b", Email: "b@x.com", Provider: "codex"},
	}}, "0.1.0")
	p.ApplyConfig(config.Config{Enabled: true, Timezone: "Asia/Taipei", Times: []string{"06:00"}, Accounts: []string{"a@x.com"}})
	defer p.Shutdown()
	h := &Handler{Plugin: p}
	resp := h.Handle(Request{Method: "GET", Path: "/v0/management/plugins/codex-selective-ping/status"})
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, resp.Body)
	}
	var st StatusResponse
	if err := json.Unmarshal(resp.Body, &st); err != nil {
		t.Fatal(err)
	}
	if st.Version == "" || st.Model == "" || st.Timezone != "Asia/Taipei" {
		t.Fatalf("%#v", st)
	}
	found := false
	for _, a := range st.Accounts {
		if a.AuthIndex == "1" {
			found = true
			if !a.Selected || a.Plan != "Plus" {
				t.Fatalf("%#v", a)
			}
		}
		if a.AuthIndex == "2" && a.Selected {
			t.Fatal("b should not be selected")
		}
	}
	if !found {
		t.Fatal("missing account 1")
	}
}

func TestRunAccepted202(t *testing.T) {
	p := plugin.New(&mh{files: nil}, "0.1.0")
	p.ApplyConfig(config.Config{Enabled: true, Timezone: "UTC", Times: []string{"06:00"}, Accounts: []string{}})
	defer p.Shutdown()
	h := &Handler{Plugin: p}
	r1 := h.Handle(Request{Method: "POST", Path: "/v0/management/plugins/codex-selective-ping/run"})
	if r1.StatusCode != 202 {
		t.Fatalf("want 202 got %d body=%s", r1.StatusCode, r1.Body)
	}
	// Wait for async empty run to finish so state cleans up.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		st := h.status()
		if !st.Running {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestRunConflict409(t *testing.T) {
	p := plugin.New(&mh{}, "0.1.0")
	p.ApplyConfig(config.Config{Enabled: true, Timezone: "UTC", Times: []string{"06:00"}, Accounts: []string{}})
	defer p.Shutdown()
	if !p.State.TryBegin() {
		t.Fatal("begin")
	}
	h := &Handler{Plugin: p}
	r := h.Handle(Request{Method: "POST", Path: "/v0/management/plugins/codex-selective-ping/run"})
	if r.StatusCode != 409 {
		t.Fatalf("got %d", r.StatusCode)
	}
}
