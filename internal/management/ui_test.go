package management

import (
	"strings"
	"testing"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/runstate"
)

func TestRenderStatusPageChineseAndQuotaDash(t *testing.T) {
	rem := 62.0
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.0", Model: "gpt-5.6-luna",
		Timezone: "Asia/Taipei", Times: []string{"06:00", "21:00"},
		Accounts: []runstate.AccountView{
			{AuthIndex: "1", Name: "alice", Email: "a@x.com", Selected: true, Plan: "Plus", FiveHour: &hostapi.QuotaWindow{Remaining: &rem}, Status: "success"},
			{AuthIndex: "2", Name: "bob", Selected: false, Status: "unknown"},
		},
	})
	for _, want := range []string{"概況", "排程", "帳號", "立刻執行", "儲存設定", "Management Key", "Plan", "5h", "週限", "—", "plugins/codex-selective-ping/config", "plugins/codex-selective-ping/run"} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q", want)
		}
	}
	if strings.Contains(html, "localStorage") {
		t.Fatal("must not use localStorage for key")
	}
}

func TestRenderStatusPageEnabledCheckbox(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.0", Model: "gpt-5.6-luna",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
	})
	if !strings.Contains(html, `id="enabled"`) {
		t.Fatal("missing enabled checkbox control")
	}
	if !strings.Contains(html, "啟用") {
		t.Fatal("missing 繁中 label for enabled")
	}
	// Must not hardcode enabled:true on save; read from checkbox instead.
	if strings.Contains(html, "enabled:true") {
		t.Fatal("save must not hardcode enabled:true")
	}
	if !strings.Contains(html, `getElementById('enabled')`) && !strings.Contains(html, `getElementById("enabled")`) {
		t.Fatal("save must read enabled from checkbox")
	}
	// Find enabled checkbox is checked when Enabled=true
	idx := strings.Index(html, `id="enabled"`)
	snippet := html[idx : idx+80]
	if !strings.Contains(snippet, "checked") {
		t.Fatalf("enabled checkbox should be checked when st.Enabled=true; snippet=%q", snippet)
	}

	htmlOff := RenderStatusPage(StatusResponse{
		Enabled: false, Version: "0.1.0", Model: "gpt-5.6-luna",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
	})
	idx = strings.Index(htmlOff, `id="enabled"`)
	if idx < 0 {
		t.Fatal("missing enabled when disabled")
	}
	snippet = htmlOff[idx : idx+80]
	if strings.Contains(snippet, "checked") {
		t.Fatalf("enabled checkbox must not be checked when st.Enabled=false; snippet=%q", snippet)
	}
}

func TestPreferIDAuthIndexFirst(t *testing.T) {
	if got := preferID(runstate.AccountView{AuthIndex: "idx-7", Email: "a@x.com", Name: "alice"}); got != "idx-7" {
		t.Fatalf("prefer auth_index when present: got %q", got)
	}
	if got := preferID(runstate.AccountView{Email: "a@x.com", Name: "alice"}); got != "a@x.com" {
		t.Fatalf("prefer email when no auth_index: got %q", got)
	}
	if got := preferID(runstate.AccountView{Name: "alice"}); got != "alice" {
		t.Fatalf("prefer name last: got %q", got)
	}
}

func TestRenderStatusPageCheckboxUsesAuthIndex(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.0", Model: "gpt-5.6-luna",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
		Accounts: []runstate.AccountView{
			{AuthIndex: "auth-42", Name: "alice", Email: "a@x.com", Selected: true},
		},
	})
	if !strings.Contains(html, `data-id="auth-42"`) {
		t.Fatalf("checkbox data-id must use auth_index when present; html snippet missing")
	}
	if strings.Contains(html, `data-id="a@x.com"`) {
		t.Fatal("checkbox data-id must not prefer email over auth_index")
	}
}
