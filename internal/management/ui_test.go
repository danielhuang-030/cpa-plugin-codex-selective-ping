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
	for _, want := range []string{"今天的節奏", "要打誰", "操作原則", "立刻執行", "儲存設定", "Management Key", "Plan", "5h", "週限", "—", "plugins/codex-selective-ping/config", "plugins/codex-selective-ping/run", "排程"} {
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

func TestRenderStatusPageRhythmTimeline(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Timezone: "Asia/Taipei",
		Enabled:  true,
		Times:    []string{"06:00", "11:00", "16:00", "21:00"},
		NextRun:  "21:00",
	}, LangZhHant)
	if !strings.Contains(html, `class="timeline"`) {
		t.Fatal("missing timeline")
	}
	if !strings.Contains(html, `class="slot next"`) && !strings.Contains(html, `class="slot next `) {
		t.Fatal("missing next slot highlight")
	}
	if !strings.Contains(html, `id="schedule_enabled"`) {
		t.Fatal("schedule_enabled must remain")
	}
}

func TestRenderStatusPageAccountCards(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.5", Model: "gpt-5.4",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
		Accounts: []runstate.AccountView{
			{AuthIndex: "1", Name: "alice", Email: "a@x.com", Selected: true, Plan: "Plus", Status: "success"},
			{AuthIndex: "2", Name: "bob", Selected: false, Status: "unknown"},
		},
		AccountsConfig: []string{"1"},
	}, LangZhHant)
	if !strings.Contains(html, `class="account-grid"`) {
		t.Fatal("missing account-grid")
	}
	if !strings.Contains(html, `class="acct`) {
		t.Fatal("missing acct card")
	}
	if !strings.Contains(html, `class="acct selected"`) && !strings.Contains(html, `class="acct selected `) {
		t.Fatal("missing selected acct card")
	}
}

func TestRenderStatusPageAccountsEmptyState(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.5", Model: "gpt-5.4",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
		Accounts: []runstate.AccountView{
			{AuthIndex: "1", Name: "alice", Selected: false},
		},
		AccountsConfig: []string{},
	}, LangZhHant)
	if !strings.Contains(html, `id="sec-accounts-empty"`) && !strings.Contains(html, `data-i18n="accounts_empty_title"`) {
		t.Fatal("missing empty whitelist empty-state")
	}
}

func TestRenderStatusPageRailActionsOrder(t *testing.T) {
	html := RenderStatusPage(StatusResponse{}, LangZhHant)
	key := strings.Index(html, `id="management-key"`)
	save := strings.Index(html, `onclick="saveCfg()"`)
	run := strings.Index(html, `onclick="runNow()"`)
	if key < 0 || save < 0 || run < 0 {
		t.Fatal("missing key/save/run controls")
	}
	if !(key < save && save < run) {
		t.Fatalf("expected key then save then run order; key=%d save=%d run=%d", key, save, run)
	}
	rail := strings.Index(html, `class="rail"`)
	ws := strings.Index(html, `class="workspace"`)
	if rail < 0 || ws < 0 || !(rail < key && key < ws && save < ws && run < ws) {
		t.Fatal("actions should live in rail")
	}
}

func TestRenderStatusPageLastRunReceipt(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.5", Model: "gpt-5.4",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
		LastRun: &runstate.Summary{
			Mode: "manual", Succeeded: 1, Limited: 1, Skipped: 2, Failed: 0,
			Accounts: []runstate.AccountResult{
				{Name: "alice", Status: "success", HTTPStatus: 200},
			},
		},
	}, LangZhHant)
	if !strings.Contains(html, `class="run-summary"`) {
		t.Fatal("missing run-summary receipt")
	}
	if !strings.Contains(html, `class="run"`) {
		t.Fatal("missing run layout")
	}
}

func TestRenderStatusPageLastRunEmpty(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.5", Model: "gpt-5.4",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
	}, LangZhHant)
	if !strings.Contains(html, `id="sec-last-empty"`) && !strings.Contains(html, `data-i18n="last_run_empty_title"`) {
		t.Fatal("missing empty last-run state")
	}
}

func TestRenderStatusPageWarmCSSTokens(t *testing.T) {
	html := RenderStatusPage(StatusResponse{Version: "0.1.5"}, LangZhHant)
	for _, want := range []string{
		"--accent:",
		`:root[data-theme="dark"]`,
		"--bg: #f6f1ea",
		"class=\"shell\"",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("warm CSS / landmark missing %q", want)
		}
	}
}
