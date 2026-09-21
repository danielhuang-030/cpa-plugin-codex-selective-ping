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
	}, LangZhHant)
	for _, want := range []string{"概況", "排程", "帳號", "立刻執行", "儲存設定", "Management Key", "Plan", "5h", "週限", "—", "plugins/codex-selective-ping/config", "plugins/codex-selective-ping/run"} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q", want)
		}
	}
	if strings.Contains(html, "localStorage.setItem") {
		t.Fatal("must not persist Management Key via localStorage.setItem")
	}
	if !strings.Contains(html, `type="password"`) {
		t.Fatal("Management Key field must be type=password")
	}
	if !strings.Contains(html, "不持久化") {
		t.Fatal("zh-Hant page must note key is not persisted")
	}
}

func TestRenderStatusPageEnabledCheckbox(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.0", Model: "gpt-5.6-luna",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
	}, LangZhHant)
	if !strings.Contains(html, `id="schedule_enabled"`) {
		t.Fatal("missing schedule_enabled checkbox control")
	}
	if !strings.Contains(html, "啟用") {
		t.Fatal("missing 繁中 label for schedule enable")
	}
	if strings.Contains(html, `id="enabled"`) {
		t.Fatal("must not use id=enabled (conflicts with CPA host lifecycle)")
	}
	if !strings.Contains(html, `getElementById('schedule_enabled')`) && !strings.Contains(html, `getElementById("schedule_enabled")`) {
		t.Fatal("save must read schedule_enabled from checkbox")
	}
	idx := strings.Index(html, `id="schedule_enabled"`)
	snippet := html[idx : idx+80]
	if !strings.Contains(snippet, "checked") {
		t.Fatalf("schedule_enabled checkbox should be checked when st.Enabled=true; snippet=%q", snippet)
	}

	htmlOff := RenderStatusPage(StatusResponse{
		Enabled: false, Version: "0.1.0", Model: "gpt-5.6-luna",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
	}, LangZhHant)
	idx = strings.Index(htmlOff, `id="schedule_enabled"`)
	if idx < 0 {
		t.Fatal("missing schedule_enabled when disabled")
	}
	snippet = htmlOff[idx : idx+80]
	if strings.Contains(snippet, "checked") {
		t.Fatalf("schedule_enabled checkbox must not be checked when st.Enabled=false; snippet=%q", snippet)
	}
}

func TestRenderStatusPageSaveUsesScheduleEnabledNotEnabled(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.0", Model: "gpt-5.6-luna",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
	}, LangEn)
	if !strings.Contains(html, `id="schedule_enabled"`) {
		t.Fatal("missing id=schedule_enabled")
	}
	if strings.Contains(html, `getElementById('enabled')`) || strings.Contains(html, `getElementById("enabled")`) {
		t.Fatal("must not call getElementById('enabled')")
	}
	// Save body object must use schedule_enabled key, not enabled
	if !strings.Contains(html, "schedule_enabled:") && !strings.Contains(html, "schedule_enabled :") {
		// JS object shorthand: schedule_enabled:document.getElementById(...)
		t.Fatal("save body must include schedule_enabled key")
	}
	// Reject bare enabled key in save body (schedule_enabled is OK; substring "enabled:" alone is too broad).
	if strings.Contains(html, "{enabled:") || strings.Contains(html, "{enabled :") || strings.Contains(html, ",enabled:") || strings.Contains(html, ", enabled:") {
		t.Fatal("save JSON construction must not include enabled key")
	}
	if strings.Contains(html, "body={enabled:") || strings.Contains(html, "body = {enabled:") {
		t.Fatal("save body must not start with enabled key")
	}
}

func TestRenderStatusPageThemeSyncJS(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.0", Model: "gpt-5.6-luna",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
	}, LangEn)
	for _, want := range []string{
		`cli-proxy-theme`,
		`data-theme`,
		`prefers-color-scheme`,
		`theme=`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("theme sync JS missing %q", want)
		}
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
	}, LangZhHant)
	if !strings.Contains(html, `data-id="auth-42"`) {
		t.Fatalf("checkbox data-id must use auth_index when present; html snippet missing")
	}
	if strings.Contains(html, `data-id="a@x.com"`) {
		t.Fatal("checkbox data-id must not prefer email over auth_index")
	}
}

func TestRenderStatusPageAuthFilesQuotaEnrichJS(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.4", Model: "gpt-5.4",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
		Accounts: []runstate.AccountView{
			{AuthIndex: "auth-1", Name: "alice", Email: "a@x.com", Selected: true},
		},
	}, LangZhHant)
	for _, want := range []string{
		`data-auth-index="auth-1"`,
		`data-col="plan"`,
		`data-col="five_hour"`,
		`data-col="weekly"`,
		`/v0/management/auth-files`,
		`/v0/management/api-call`,
		`backend-api/wham/usage`,
		`enrichQuotaFromManagement`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("status page JS/markup missing %q", want)
		}
	}
	// Without management key the page must keep em-dash placeholders (no invented numbers in static HTML).
	if !strings.Contains(html, "—") {
		t.Fatal("expected em-dash placeholders when quota unknown")
	}
}

func TestRenderStatusPageV3ShellLandmarks(t *testing.T) {
	html := RenderStatusPage(StatusResponse{Version: "0.1.5"}, LangZhHant)
	for _, needle := range []string{`class="shell"`, `class="rail"`, `class="workspace"`} {
		if !strings.Contains(html, needle) {
			t.Fatalf("missing v3 landmark %s", needle)
		}
	}
}
