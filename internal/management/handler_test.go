package management

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"cpa-plugin-codex-selective-ping/internal/config"
	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/pinger"
	"cpa-plugin-codex-selective-ping/internal/plugin"
)

type mh struct {
	files []hostapi.AuthFile
	creds map[string][]byte
}

func (m *mh) AuthList(context.Context) ([]hostapi.AuthFile, error) { return m.files, nil }
func (m *mh) AuthGet(_ context.Context, authIndex string) ([]byte, error) {
	if m.creds != nil {
		if raw, ok := m.creds[authIndex]; ok {
			return raw, nil
		}
	}
	return json.Marshal(map[string]any{"access_token": "t"})
}
func (m *mh) HTTPDo(context.Context, hostapi.HTTPRequest) (hostapi.HTTPResponse, error) {
	return hostapi.HTTPResponse{StatusCode: 200}, nil
}
func (m *mh) AuthGetRuntime(context.Context, string) (json.RawMessage, error) {
	return nil, hostapi.ErrUnsupported
}

func fakeIDTokenJWT(plan string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payloadObj := map[string]any{
		"email": "a@x.com",
		"https://api.openai.com/auth": map[string]any{"chatgpt_plan_type": plan},
	}
	raw, _ := json.Marshal(payloadObj)
	payload := base64.RawURLEncoding.EncodeToString(raw)
	return header + "." + payload + ".sig"
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

func TestStatusJSONUsesScheduleEnabled(t *testing.T) {
	p := plugin.New(&mh{files: nil}, "0.1.0")
	p.ApplyConfig(config.Config{Enabled: false, Timezone: "UTC", Times: []string{"06:00"}, Accounts: []string{}})
	defer p.Shutdown()
	h := &Handler{Plugin: p}
	resp := h.Handle(Request{Method: "GET", Path: "/v0/management/plugins/codex-selective-ping/status"})
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, resp.Body)
	}
	var raw map[string]any
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["schedule_enabled"]; !ok {
		t.Fatalf("status JSON must expose schedule_enabled; keys=%v", raw)
	}
	if _, ok := raw["enabled"]; ok {
		t.Fatal("status JSON must not expose enabled (host lifecycle confusion)")
	}
	if raw["schedule_enabled"] != false {
		t.Fatalf("schedule_enabled=%v want false", raw["schedule_enabled"])
	}
}

func TestStatusEnrichesPlanFromAuthGet(t *testing.T) {
	tok := fakeIDTokenJWT("plus")
	cred, _ := json.Marshal(map[string]any{"access_token": "t", "id_token": tok})
	p := plugin.New(&mh{
		files: []hostapi.AuthFile{
			{AuthIndex: "1", Name: "a", Email: "a@x.com", Provider: "codex"},
		},
		creds: map[string][]byte{"1": cred},
	}, "0.1.0")
	p.ApplyConfig(config.Config{Enabled: true, Timezone: "UTC", Times: []string{"06:00"}, Accounts: []string{"1"}})
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
	if len(st.Accounts) != 1 {
		t.Fatalf("accounts=%d", len(st.Accounts))
	}
	if st.Accounts[0].Plan != "plus" {
		t.Fatalf("plan=%q want plus (from AuthGet id_token)", st.Accounts[0].Plan)
	}
}

func TestStatusJSONIncludesAccountTimes(t *testing.T) {
	p := plugin.New(&mh{files: nil}, "0.1.0")
	p.ApplyConfig(config.Config{
		Enabled:  true,
		Timezone: "UTC",
		Times:    []string{"06:00"},
		Accounts: []string{"alice@example.com", "bob@example.com"},
		AccountTimes: map[string][]string{
			"bob@example.com": {"07:30", "19:00"},
		},
	})
	defer p.Shutdown()
	h := &Handler{Plugin: p}
	resp := h.Handle(Request{Method: "GET", Path: "/v0/management/plugins/codex-selective-ping/status"})
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, resp.Body)
	}
	var raw map[string]any
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		t.Fatal(err)
	}
	at, ok := raw["account_times"]
	if !ok {
		t.Fatalf("status JSON must include account_times when set; keys=%v", raw)
	}
	m, ok := at.(map[string]any)
	if !ok {
		t.Fatalf("account_times type=%T want object", at)
	}
	bob, ok := m["bob@example.com"].([]any)
	if !ok {
		t.Fatalf("bob times=%#v", m["bob@example.com"])
	}
	if len(bob) != 2 || bob[0] != "07:30" || bob[1] != "19:00" {
		t.Fatalf("bob times=%v want [07:30 19:00]", bob)
	}
}

func TestStatusJSONOmitsEmptyAccountTimes(t *testing.T) {
	p := plugin.New(&mh{files: nil}, "0.1.0")
	p.ApplyConfig(config.Config{
		Enabled:  true,
		Timezone: "UTC",
		Times:    []string{"06:00"},
		Accounts: []string{"alice@example.com"},
	})
	defer p.Shutdown()
	h := &Handler{Plugin: p}
	resp := h.Handle(Request{Method: "GET", Path: "/v0/management/plugins/codex-selective-ping/status"})
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, resp.Body)
	}
	var raw map[string]any
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["account_times"]; ok {
		t.Fatalf("account_times must be omitted when empty; got %#v", raw["account_times"])
	}
}

func TestStatusJSONShowsEffectiveModel(t *testing.T) {
	p := plugin.New(&mh{files: nil}, "0.1.0")
	p.ApplyConfig(config.Config{
		Enabled:  true,
		Timezone: "UTC",
		Times:    []string{"06:00"},
		Accounts: []string{},
		Model:    "gpt-5",
	})
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
	if st.Model != "gpt-5" {
		t.Fatalf("model=%q want gpt-5 (cfg.Model)", st.Model)
	}

	htmlResp := h.Handle(Request{Method: "GET", Path: "/v0/resource/plugins/codex-selective-ping/status"})
	if htmlResp.StatusCode != 200 {
		t.Fatalf("html %d %s", htmlResp.StatusCode, htmlResp.Body)
	}
	if !strings.Contains(string(htmlResp.Body), "gpt-5") {
		t.Fatal("status HTML must show cfg.Model gpt-5")
	}
}

func TestStatusJSONEmptyModelFallsBack(t *testing.T) {
	p := plugin.New(&mh{files: nil}, "0.1.0")
	p.ApplyConfig(config.Config{
		Enabled:  true,
		Timezone: "UTC",
		Times:    []string{"06:00"},
		Accounts: []string{},
		Model:    "",
	})
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
	if st.Model != pinger.ModelName {
		t.Fatalf("model=%q want %q (pinger.ModelName fallback)", st.Model, pinger.ModelName)
	}
}

func TestModelsEndpointManagementKeyOK(t *testing.T) {
	keysBody, _ := json.Marshal(map[string]any{"api-keys": []string{"sk-proxy"}})
	body, _ := json.Marshal(map[string]any{
		"data": []map[string]any{
			{"id": "gpt-6-luna", "owned_by": "openai"},
			{"id": "claude-3", "owned_by": "anthropic"},
		},
	})
	host := &modelsHost{script: []hostapi.HTTPResponse{
		{StatusCode: 200, Body: keysBody},
		{StatusCode: 200, Body: body},
	}}
	p := plugin.New(host, "0.2.5")
	p.ApplyConfig(config.Config{Enabled: true, Model: "gpt-6-luna"})
	defer p.Shutdown()
	h := &Handler{Plugin: p}
	resp := h.Handle(Request{
		Method: "GET",
		Path:   "/v0/management/plugins/codex-selective-ping/models",
		Headers: map[string][]string{
			"Authorization": {"Bearer mgmt-key"},
			"Host":          {"cpa.test"},
		},
	})
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d body=%s", resp.StatusCode, resp.Body)
	}
	var out modelsListResponse
	if err := json.Unmarshal(resp.Body, &out); err != nil {
		t.Fatal(err)
	}
	if out.Warning != "" {
		t.Fatalf("warning=%q", out.Warning)
	}
	ids := modelIDs(out)
	if !hasID(ids, "gpt-6-luna") || hasID(ids, "claude-3") {
		t.Fatalf("ids=%v", ids)
	}
	if len(host.calls) != 2 {
		t.Fatalf("calls=%v", host.calls)
	}
	if host.calls[0].URL != "http://cpa.test/v0/management/api-keys" {
		t.Fatalf("keys url=%q", host.calls[0].URL)
	}
	if host.calls[1].URL != "http://cpa.test/v1/models" {
		t.Fatalf("models url=%q", host.calls[1].URL)
	}
	if firstHeader(host.calls[1].Headers, "Authorization") != "Bearer sk-proxy" {
		t.Fatalf("models auth=%q", firstHeader(host.calls[1].Headers, "Authorization"))
	}
}

func TestModelsEndpointFallbackOnUpstream401(t *testing.T) {
	host := &modelsHost{
		files: []hostapi.AuthFile{{AuthIndex: "1", Provider: "codex"}},
		script: []hostapi.HTTPResponse{
			{StatusCode: 401, Body: []byte(`{}`)},
			{StatusCode: 401, Body: []byte(`{}`)},
		},
	}
	p := plugin.New(host, "0.2.5")
	p.ApplyConfig(config.Config{Enabled: true, Model: "my-custom"})
	defer p.Shutdown()
	h := &Handler{Plugin: p}
	resp := h.Handle(Request{
		Method: "GET",
		Path:   "/v0/management/plugins/codex-selective-ping/models",
		Headers: map[string][]string{
			"Authorization": {"Bearer mgmt-key"},
			"Host":          {"cpa.test"},
		},
	})
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d body=%s", resp.StatusCode, resp.Body)
	}
	var out modelsListResponse
	if err := json.Unmarshal(resp.Body, &out); err != nil {
		t.Fatal(err)
	}
	if out.Warning == "" {
		t.Fatal("expected warning")
	}
	ids := modelIDs(out)
	if !hasID(ids, "my-custom") || !hasID(ids, pinger.ModelName) {
		t.Fatalf("ids=%v", ids)
	}
}
