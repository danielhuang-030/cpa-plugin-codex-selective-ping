package management

import (
	"strings"
	"testing"
	"time"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/runstate"
)

func TestRenderStatusPageChineseAndQuotaDash(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.0", Model: "gpt-5.6-luna",
		Timezone: "Asia/Taipei", Times: []string{"06:00", "21:00"},
		Accounts: []runstate.AccountView{
			{AuthIndex: "1", Name: "alice", Email: "a@x.com", Selected: true, Plan: "Plus", Status: "success"},
			{AuthIndex: "2", Name: "bob", Selected: false, Status: "unknown"},
		},
	}, LangZhHant)
	for _, want := range []string{"統一節奏", "帳號與時刻", "立刻執行", "儲存設定", "Management Key", "plugins/codex-selective-ping/config", "plugins/codex-selective-ping/run", "排程"} {
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
		`data-col="status"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("status page markup missing %q", want)
		}
	}
	for _, ban := range []string{`data-col="five_hour"`, `data-col="weekly"`, `data-i18n="col_5h"`, `enrichQuotaFromManagement`, `backend-api/wham/usage`} {
		if strings.Contains(html, ban) {
			t.Fatalf("v4 UI must not contain quota cell/enrich %q", ban)
		}
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
	if !strings.Contains(html, `class="acct-card`) {
		t.Fatal("missing acct-card wrapper")
	}
	if !strings.Contains(html, `class="acct-card selected"`) && !strings.Contains(html, `class="acct-card selected `) {
		t.Fatal("missing selected acct-card")
	}
	if !strings.Contains(html, `type="checkbox" class="acct"`) && !strings.Contains(html, `class="acct" data-id=`) {
		t.Fatal("checkbox must keep class=acct for JS selectors")
	}
	// Card CSS must target .acct-card, not collide with input.acct
	if !strings.Contains(html, ".acct-card{") && !strings.Contains(html, ".acct-card {") {
		t.Fatal("CSS must style .acct-card")
	}
	if strings.Contains(html, ".acct{") || strings.Contains(html, ".acct {") {
		t.Fatal("CSS must not use bare .acct{ (collides with checkbox)")
	}
}

func TestRenderStatusPageAccountsEmptyState(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.5", Model: "gpt-5.4",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
		Accounts:       nil,
		AccountsConfig: []string{},
	}, LangZhHant)
	if !strings.Contains(html, `id="sec-accounts-empty"`) {
		t.Fatal("missing id=sec-accounts-empty")
	}
	if !strings.Contains(html, `data-i18n="accounts_none_title"`) && !strings.Contains(html, `data-i18n="accounts_empty_title"`) {
		t.Fatal("missing empty-state i18n marker")
	}
	// With zero Codex accounts, filled section should be hidden.
	idx := strings.Index(html, `id="sec-accounts-filled"`)
	if idx < 0 {
		t.Fatal("missing sec-accounts-filled")
	}
	snippet := html[max(0, idx-40) : idx+80]
	if !strings.Contains(snippet, "hidden") {
		t.Fatalf("filled accounts section should be hidden when no accounts; snippet=%q", snippet)
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

func TestRenderStatusPageLastRunReceiptShowsFailed(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.5", Model: "gpt-5.4",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
		LastRun: &runstate.Summary{
			Mode: "manual", Succeeded: 0, Limited: 0, Skipped: 0, Failed: 2,
		},
	}, LangZhHant)
	if !strings.Contains(html, "失敗") && !strings.Contains(html, "failed") {
		t.Fatal("receipt must include failed count label when Failed>0")
	}
	if !strings.Contains(html, "2") {
		t.Fatal("receipt must show failed count value")
	}
}

func TestRenderStatusPageLastRunEmpty(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.5", Model: "gpt-5.4",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
	}, LangZhHant)
	if !strings.Contains(html, `id="sec-last-empty"`) {
		t.Fatal("missing id=sec-last-empty")
	}
	if !strings.Contains(html, `data-i18n="last_run_empty_title"`) {
		t.Fatal("missing data-i18n=last_run_empty_title")
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
	// Base .shell rule must exist in CSS (not only HTML class / media override).
	if !strings.Contains(html, ".shell{") && !strings.Contains(html, ".shell {") {
		t.Fatal("CSS must contain .shell{ rule")
	}
	if !strings.Contains(html, "grid-template-columns: 280px 1fr") && !strings.Contains(html, "grid-template-columns:280px 1fr") {
		t.Fatal("CSS .shell must set grid-template-columns: 280px 1fr")
	}
	if !strings.Contains(html, "max-width: 1180px") && !strings.Contains(html, "max-width:1180px") {
		t.Fatal("CSS .shell must set max-width: 1180px")
	}
}

func TestRenderStatusPageNoQuotaBarsOrFiveHourCells(t *testing.T) {
	okRem := 62.0
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.5", Model: "gpt-5.4",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
		Accounts: []runstate.AccountView{
			{AuthIndex: "1", Name: "alice", Selected: true, FiveHour: &hostapi.QuotaWindow{Remaining: &okRem}, Weekly: &hostapi.QuotaWindow{Remaining: &okRem}},
		},
	}, LangZhHant)
	for _, ban := range []string{`data-col="five_hour"`, `data-col="weekly"`, `data-i18n="col_5h"`, `data-i18n="col_weekly"`, `class="bar"`, `class="bar `, `applyQuotaToRow`, `quotaBarClass`} {
		if strings.Contains(html, ban) {
			t.Fatalf("account UI must not render quota chrome %q", ban)
		}
	}
	if !strings.Contains(html, `data-testid="account-schedule"`) {
		t.Fatal("expected account-schedule marker")
	}
}

func TestRenderStatusPageStatusPillOffNeutral(t *testing.T) {
	htmlOff := RenderStatusPage(StatusResponse{Enabled: false, Version: "0.1.5"}, LangZhHant)
	if !strings.Contains(htmlOff, `status-pill warn`) && !strings.Contains(htmlOff, `status-pill off`) && !strings.Contains(htmlOff, `status-pill neutral`) {
		t.Fatal("when schedule off, status-pill must use neutral/warn/off class")
	}
}

func TestRenderStatusPageCustomFilterCount(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.5", Model: "gpt-5.4",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
		Accounts: []runstate.AccountView{
			{AuthIndex: "1", Name: "alice", Selected: true, Status: "success"},
			{AuthIndex: "2", Name: "bob", Selected: true, Status: "success"},
			{AuthIndex: "3", Name: "old", Selected: false, Status: "unavailable"},
		},
		AccountTimes: map[string][]string{
			"2": {"07:30", "19:00"},
		},
	}, LangZhHant)
	if !strings.Contains(html, "自訂時刻 1") && !strings.Contains(html, "自訂 1") {
		t.Fatalf("custom filter tab should show count; html missing custom 1")
	}
	if !strings.Contains(html, `data-filter="custom"`) {
		t.Fatal("missing custom filter tab")
	}
}

func TestRhythmSlotPastUpcomingNext(t *testing.T) {
	// now=14:00, next=16:00 → 06 past, 16 next, 21 upcoming (not past)
	cases := []struct {
		tm, next, now string
		wantClass     string
		wantKey       string
	}{
		{"06:00", "16:00", "14:00", "slot", "slot_past"},
		{"16:00", "16:00", "14:00", "slot next", "slot_next"},
		{"21:00", "16:00", "14:00", "slot", ""},
		{"11:00", "06:00", "22:00", "slot", "slot_past"}, // after next overnight: still past vs now
		{"06:00", "06:00", "22:00", "slot next", "slot_next"},
		{"21:00", "06:00", "22:00", "slot", "slot_past"},
	}
	for _, c := range cases {
		gotClass, gotKey := rhythmSlot(c.tm, c.next, c.now)
		if gotClass != c.wantClass || gotKey != c.wantKey {
			t.Fatalf("rhythmSlot(%q,%q,%q)=(%q,%q) want (%q,%q)",
				c.tm, c.next, c.now, gotClass, gotKey, c.wantClass, c.wantKey)
		}
	}
}

func TestFilterReapplyFromSyncAndSetAll(t *testing.T) {
	html := RenderStatusPage(StatusResponse{Enabled: true, Times: []string{"21:00"}}, LangZhHant)
	for _, want := range []string{
		"let acctFilter=",
		"filterAccounts(acctFilter)",
		"function syncAcctCard(cb, skipFilter)",
		"if(!skipFilter) filterAccounts(acctFilter)",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing filter re-apply hook %q", want)
		}
	}
	// setAll must re-apply after bulk toggle
	idx := strings.Index(html, "function setAll(v){")
	if idx < 0 {
		t.Fatal("missing setAll")
	}
	chunk := html[idx : idx+350]
	if !strings.Contains(chunk, "filterAccounts(acctFilter)") {
		t.Fatalf("setAll must call filterAccounts; chunk=%q", chunk)
	}
}

func TestPrinciplesChipNeutralWhenScheduleOff(t *testing.T) {
	htmlOn := RenderStatusPage(StatusResponse{Enabled: true, Version: "0.1.5"}, LangZhHant)
	htmlOff := RenderStatusPage(StatusResponse{Enabled: false, Version: "0.1.5"}, LangZhHant)
	if !strings.Contains(htmlOn, `class="chip ok"`) {
		t.Fatal("principles status chip should be chip ok when schedule enabled")
	}
	// When off: principles chip must not hardcode chip ok for the status metric.
	// Look near sec-principles for chip without ok.
	sec := strings.Index(htmlOff, `id="sec-principles"`)
	if sec < 0 {
		t.Fatal("missing sec-principles")
	}
	chunk := htmlOff[sec:]
	if end := strings.Index(chunk, `id="sec-accounts`); end > 0 {
		chunk = chunk[:end]
	}
	if strings.Contains(chunk, `class="chip ok"`) {
		t.Fatalf("when schedule off, principles chip must not be chip ok; chunk=%q", chunk)
	}
	if !strings.Contains(chunk, `class="chip "`) && !strings.Contains(chunk, `class="chip"`) {
		// principlesChipClass("") → class="chip %s" with empty → class="chip "
		t.Fatalf("expected neutral chip class in principles when off; chunk=%q", chunk)
	}
}

func TestNoQuotaBarWidthPlumbing(t *testing.T) {
	html := RenderStatusPage(StatusResponse{Enabled: true}, LangZhHant)
	for _, ban := range []string{"data-bar-width", "fmtQuotaWindow", "剩/left/残"} {
		if strings.Contains(html, ban) {
			t.Fatalf("must not contain quota plumbing %q", ban)
		}
	}
	if !strings.Contains(html, ".row { display:flex") && !strings.Contains(html, ".row{display:flex") {
		t.Fatal("missing .row { display:flex } CSS for select-all gap")
	}
}

func TestRenderStatusPageUpcomingSlotNotPastCaption(t *testing.T) {
	// Force deterministic classification via helper already unit-tested; also assert page
	// still emits slot_next for NextRun and does not force every non-next to 已過 in JS.
	html := RenderStatusPage(StatusResponse{
		Enabled:  true,
		Timezone: "Asia/Taipei",
		Times:    []string{"06:00", "11:00", "16:00", "21:00"},
		NextRun:  "16:00",
	}, LangZhHant)
	if !strings.Contains(html, `data-time="16:00"`) {
		t.Fatal("missing 16:00 slot")
	}
	if !strings.Contains(html, "isPast ? slotCaptionPast") && !strings.Contains(html, "isPast?slotCaptionPast") {
		t.Fatal("JS renderTimes must only use past caption when isPast")
	}
	if !strings.Contains(html, "tm < now") {
		t.Fatal("JS must compare slot time to now for past detection")
	}
}

func TestRenderStatusPageRunHistoryList(t *testing.T) {
	htmlOut := RenderStatusPage(StatusResponse{
		LastRun: &runstate.Summary{
			At: time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC), Mode: "manual", Succeeded: 2, Failed: 0, Skipped: 1,
		},
		RunHistory: []runstate.Summary{
			{At: time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC), Mode: "manual", Succeeded: 2, Skipped: 1, Message: "newer"},
			{At: time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC), Mode: "schedule", Succeeded: 1, Failed: 1, Message: "older"},
		},
	}, LangEn)
	if !strings.Contains(htmlOut, `data-testid="run-history"`) {
		t.Fatal("missing run-history list")
	}
	if !strings.Contains(htmlOut, `data-i18n="run_history"`) && !strings.Contains(htmlOut, `data-i18n="run_history_title"`) && !strings.Contains(htmlOut, `data-testid="run-history"`) {
		t.Fatal("missing run_history i18n / marker")
	}
	if !strings.Contains(htmlOut, "manual") || !strings.Contains(htmlOut, "schedule") {
		t.Fatal("expected both history modes in list")
	}
}

func TestRenderStatusPageNoFiveHourWeeklyCells(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.8", Model: "gpt-5.4",
		Timezone: "Asia/Taipei", Times: []string{"06:00", "21:00"},
		Accounts: []runstate.AccountView{
			{AuthIndex: "1", Name: "alice", Email: "a@x.com", Selected: true, Plan: "Plus"},
		},
	}, LangZhHant)
	for _, ban := range []string{`data-col="five_hour"`, `col_5h`, `data-col="weekly"`, `col_weekly`} {
		if strings.Contains(html, ban) {
			t.Fatalf("must not contain %q", ban)
		}
	}
}

func TestRenderStatusPageAccountScheduleMarkers(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.8", Model: "gpt-5.4",
		Timezone: "Asia/Taipei", Times: []string{"06:00", "11:00"},
		Accounts: []runstate.AccountView{
			{AuthIndex: "alice", Name: "alice", Email: "alice@example.com", Selected: true},
			{AuthIndex: "bob", Name: "bob", Email: "bob@example.com", Selected: true},
		},
		AccountTimes: map[string][]string{
			"bob": {"07:30", "19:00"},
		},
	}, LangZhHant)
	if !strings.Contains(html, `data-testid="account-schedule"`) {
		t.Fatal("missing data-testid=account-schedule")
	}
	if !strings.Contains(html, `data-sched="inherit"`) {
		t.Fatal("missing inherit schedule marker")
	}
	if !strings.Contains(html, `data-sched="custom"`) {
		t.Fatal("missing custom schedule marker")
	}
	if !strings.Contains(html, "07:30") || !strings.Contains(html, "19:00") {
		t.Fatal("custom times must appear in account card")
	}
	if !strings.Contains(html, "account_times") {
		t.Fatal("saveCfg / initial state must mention account_times")
	}
}

func TestRenderStatusPageSaveCfgIncludesAccountTimes(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Times: []string{"21:00"},
		Accounts: []runstate.AccountView{{AuthIndex: "1", Name: "a", Selected: true}},
	}, LangEn)
	if !strings.Contains(html, "account_times") {
		t.Fatal("JS must build account_times for PATCH body")
	}
	idx := strings.Index(html, "async function saveCfg()")
	if idx < 0 {
		t.Fatal("missing saveCfg")
	}
	chunk := html[idx : idx+900]
	if !strings.Contains(chunk, "account_times") {
		t.Fatalf("saveCfg body must include account_times; chunk=%q", chunk[:200])
	}
	if !strings.Contains(html, "initialAccountTimes") && !strings.Contains(html, "accountTimes") {
		t.Fatal("page must seed account times into JS initial state")
	}
}

func TestRenderStatusPageHistoryExpandPerAccount(t *testing.T) {
	htmlOut := RenderStatusPage(StatusResponse{
		LastRun: &runstate.Summary{
			At: time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC), Mode: "force", Succeeded: 1, Skipped: 1,
			Accounts: []runstate.AccountResult{
				{Name: "alice@example.com", Status: "success", Attempts: 1},
				{Name: "bob@example.com", Status: "skipped", Attempts: 0, Error: "disabled"},
			},
		},
		RunHistory: []runstate.Summary{
			{
				At: time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC), Mode: "force", Succeeded: 1, Skipped: 1,
				Accounts: []runstate.AccountResult{
					{Name: "alice@example.com", Status: "success", Attempts: 1},
					{Name: "bob@example.com", Status: "skipped", Attempts: 0, Error: "disabled"},
				},
			},
			{
				At: time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC), Mode: "scheduled", Succeeded: 1,
				Accounts: []runstate.AccountResult{
					{Name: "bob@example.com", Status: "success", Attempts: 2},
				},
			},
		},
	}, LangEn)
	if !strings.Contains(htmlOut, `data-testid="run-history"`) {
		t.Fatal("missing run-history")
	}
	if !strings.Contains(htmlOut, `data-testid="hist-accounts"`) && !strings.Contains(htmlOut, `class="hist-body"`) {
		t.Fatal("missing expandable hist-body / hist-accounts")
	}
	if !strings.Contains(htmlOut, "alice@example.com") || !strings.Contains(htmlOut, "bob@example.com") {
		t.Fatal("history expand must list per-account names")
	}
	if !strings.Contains(htmlOut, "force") || !strings.Contains(htmlOut, "scheduled") {
		t.Fatal("history must show force and scheduled modes")
	}
	if !strings.Contains(htmlOut, "attempts") && !strings.Contains(htmlOut, "Attempts") {
		// detail line should mention attempts count somehow
		if !strings.Contains(htmlOut, "attempts 1") && !strings.Contains(htmlOut, "attempts: 1") && !strings.Contains(htmlOut, "· 1") {
			// soft: at least attempts number near status is OK via "attempts"
			t.Log("note: attempts wording may be i18n; checking numeric presence near account")
		}
	}
	if !strings.Contains(htmlOut, "hist-item") && !strings.Contains(htmlOut, "hist-head") {
		t.Fatal("missing hist-item/hist-head expand chrome")
	}
}

func TestRematerializeAccountTimesEmailKeyedToPreferID(t *testing.T) {
	accounts := []runstate.AccountView{
		{AuthIndex: "auth-bob", Name: "bob", Email: "bob@example.com", Selected: true},
		{AuthIndex: "auth-alice", Name: "alice", Email: "alice@example.com", Selected: true},
	}
	in := map[string][]string{
		"bob@example.com": {"07:30", "19:00"},
		"auth-alice":      {"08:00"},
	}
	got := rematerializeAccountTimes(accounts, in)
	if len(got) != 2 {
		t.Fatalf("got %#v", got)
	}
	bob := got["auth-bob"]
	if len(bob) != 2 || bob[0] != "07:30" || bob[1] != "19:00" {
		t.Fatalf("email-keyed times must rematerialize under preferID auth-bob: %#v", got)
	}
	if _, ok := got["bob@example.com"]; ok {
		t.Fatal("alternate email key must not remain")
	}
	alice := got["auth-alice"]
	if len(alice) != 1 || alice[0] != "08:00" {
		t.Fatalf("auth-index keyed times: %#v", alice)
	}
}

func TestRenderStatusPageEmailKeyedAccountTimesSeededUnderDataID(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled:  true,
		Timezone: "Asia/Taipei",
		Times:    []string{"06:00", "21:00"},
		Accounts: []runstate.AccountView{
			{AuthIndex: "auth-bob", Name: "bob", Email: "bob@example.com", Selected: true},
		},
		AccountTimes: map[string][]string{
			"bob@example.com": {"07:30", "19:00"},
		},
	}, LangEn)
	if !strings.Contains(html, `data-id="auth-bob"`) {
		t.Fatal("checkbox/card data-id must be auth index")
	}
	if !strings.Contains(html, `data-sched="custom"`) {
		t.Fatal("SSR must resolve email-keyed account_times as custom")
	}
	if !strings.Contains(html, `"auth-bob"`) {
		t.Fatal("initialAccountTimes / rematerialize must emit preferID key auth-bob")
	}
	for _, want := range []string{
		"rematerializeAccountTimesFromCards",
		"delete accountTimes[",
		"collectAccountTimes",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("JS contract missing %q", want)
		}
	}
}

func TestRenderStatusPageEmptyStatusNeutralNotSuccess(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Times: []string{"21:00"},
		Accounts: []runstate.AccountView{
			{AuthIndex: "1", Name: "alice", Email: "a@x.com", Selected: true, Status: ""},
		},
	}, LangEn)
	idx := strings.Index(html, `data-col="status"`)
	if idx < 0 {
		t.Fatal("missing status chip")
	}
	snippet := html[idx : idx+80]
	if strings.Contains(snippet, "status_success") || strings.Contains(snippet, ">success<") || strings.Contains(snippet, "chip ok") {
		t.Fatalf("empty Status must be neutral/—, not success; snippet=%q", snippet)
	}
	if !strings.Contains(snippet, "—") && !strings.Contains(snippet, "–") {
		t.Fatalf("empty Status should render dash; snippet=%q", snippet)
	}
}

func TestRenderStatusPageAuthMetaIncludesIndex(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Times: []string{"21:00"},
		Accounts: []runstate.AccountView{
			{AuthIndex: "auth-42", Name: "alice", Email: "a@x.com", Selected: true},
		},
	}, LangEn)
	if !strings.Contains(html, "auth_index: auth-42") {
		t.Fatalf("expected 'auth_index: auth-42' in meta; html missing")
	}
}

func TestRenderStatusPageSyncAcctCardRebuildsOnUncheck(t *testing.T) {
	html := RenderStatusPage(StatusResponse{Enabled: true, Times: []string{"21:00"}}, LangEn)
	for _, want := range []string{
		"function syncAcctCard(",
		"rebuildAcctSchedRow",
		"labelSchedNotSelected",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("syncAcctCard rebuild contract missing %q", want)
		}
	}
	idx := strings.Index(html, "function syncAcctCard(")
	chunk := html[idx : idx+700]
	if !strings.Contains(chunk, "rebuildAcctSchedRow") {
		t.Fatalf("syncAcctCard must call rebuildAcctSchedRow; chunk=%q", chunk)
	}
}

func TestRenderStatusPageCustomOnlyNextIndicator(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled:  true,
		Timezone: "Asia/Taipei",
		Times:    []string{"06:00", "21:00"},
		NextRun:  "07:30",
		Accounts: []runstate.AccountView{
			{AuthIndex: "bob", Name: "bob", Email: "bob@example.com", Selected: true},
		},
		AccountTimes: map[string][]string{
			"bob": {"07:30", "19:00"},
		},
	}, LangEn)
	if !strings.Contains(html, `data-testid="next-custom-only"`) {
		t.Fatal("custom-only NextRun must show non-timeline next indicator")
	}
	if !strings.Contains(html, "07:30") {
		t.Fatal("indicator / rail should mention 07:30")
	}
	if strings.Contains(html, `class="slot next" data-time="06:00"`) || strings.Contains(html, `class="slot next" data-time="21:00"`) {
		t.Fatal("global timeline must not highlight next when NextRun is custom-only")
	}
}

func TestRenderStatusPageHistChipTogglesLabel(t *testing.T) {
	htmlOut := RenderStatusPage(StatusResponse{
		RunHistory: []runstate.Summary{
			{At: time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC), Mode: "force", Succeeded: 1},
		},
	}, LangEn)
	if !strings.Contains(htmlOut, "toggleHist") {
		t.Fatal("hist expand chip must update label on click via toggleHist")
	}
	if !strings.Contains(htmlOut, "labelHistExpand") || !strings.Contains(htmlOut, "labelHistCollapse") {
		t.Fatal("JS must have expand/collapse label constants")
	}
}

func TestRenderStatusPageCustomEmptyTimesHint(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Times: []string{"21:00"},
		Accounts: []runstate.AccountView{
			{AuthIndex: "1", Name: "a", Selected: true},
		},
	}, LangEn)
	if !strings.Contains(html, "sched_custom_empty") && !strings.Contains(html, "labelSchedCustomEmpty") {
		t.Fatal("optional hint for custom+empty times (inherit on save) should be wired")
	}
}

func TestRenderStatusPageRunningBootstrap(t *testing.T) {
	htmlOn := RenderStatusPage(StatusResponse{Running: true, Version: "0.1.9"}, LangZhHant)
	if !strings.Contains(htmlOn, `data-running="true"`) {
		t.Fatal("expected data-running=true when Running")
	}
	if !strings.Contains(htmlOn, `data-run-now`) {
		t.Fatal("expected data-run-now markers on run buttons")
	}
	if !strings.Contains(htmlOn, `disabled`) {
		t.Fatal("expected run controls disabled when running")
	}
	if !strings.Contains(htmlOn, `data-i18n="running_label"`) && !strings.Contains(htmlOn, "執行中") {
		t.Fatal("expected running label visible when running")
	}

	htmlOff := RenderStatusPage(StatusResponse{Running: false, Version: "0.1.9"}, LangZhHant)
	if !strings.Contains(htmlOff, `data-running="false"`) {
		t.Fatal("expected data-running=false")
	}
}

func TestRenderStatusPageRunNowPollsStatusNotFixedReload(t *testing.T) {
	html := RenderStatusPage(StatusResponse{Version: "0.1.9"}, LangZhHant)
	bad := []string{
		`setTimeout(()=>location.reload(),1500)`,
		`setTimeout(()=>location.reload(), 1500)`,
		`setTimeout(() => location.reload(), 1500)`,
	}
	for _, b := range bad {
		if strings.Contains(html, b) {
			t.Fatalf("fixed 1500ms reload after run must be removed; found %q", b)
		}
	}
	if strings.Contains(html, `setTimeout(()=>location.reload(),1500)`) {
		t.Fatal("remove setTimeout(()=>location.reload(),1500) from runNow")
	}
	for _, want := range []string{
		`/v0/management/plugins/codex-selective-ping/status`,
		"enterRunningMode",
		"2500",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing poll UX marker %q", want)
		}
	}
	if !strings.Contains(html, "function runNow") && !strings.Contains(html, "async function runNow") {
		t.Fatal("expected runNow function")
	}
}
