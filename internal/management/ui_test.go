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
		Accounts: []runstate.AccountView{
			{AuthIndex: "1", Name: "alice", Selected: false},
		},
		AccountsConfig: []string{},
	}, LangZhHant)
	if !strings.Contains(html, `id="sec-accounts-empty"`) {
		t.Fatal("missing id=sec-accounts-empty")
	}
	if !strings.Contains(html, `data-i18n="accounts_empty_title"`) {
		t.Fatal("missing data-i18n=accounts_empty_title")
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

func TestRenderStatusPageQuotaBars(t *testing.T) {
	okRem := 62.0
	warnRem := 18.0
	badRem := 0.0
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.5", Model: "gpt-5.4",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
		Accounts: []runstate.AccountView{
			{AuthIndex: "1", Name: "alice", Selected: true, FiveHour: &hostapi.QuotaWindow{Remaining: &okRem}, Weekly: &hostapi.QuotaWindow{Remaining: &okRem}},
			{AuthIndex: "2", Name: "bob", Selected: false, FiveHour: &hostapi.QuotaWindow{Remaining: &warnRem}},
			{AuthIndex: "3", Name: "spare", Selected: false, FiveHour: &hostapi.QuotaWindow{Remaining: &badRem}},
		},
	}, LangZhHant)
	if !strings.Contains(html, `class="bar"`) && !strings.Contains(html, `class="bar `) {
		t.Fatal("missing .bar markup when quota parseable")
	}
	if !strings.Contains(html, `width:62%`) && !strings.Contains(html, `width:62.`) {
		t.Fatal("bar width should reflect remaining percent")
	}
	if !strings.Contains(html, `class="bar warn"`) {
		t.Fatal("low remaining should use bar.warn")
	}
	if !strings.Contains(html, `class="bar bad"`) {
		t.Fatal("zero remaining should use bar.bad")
	}
	// Enrich path must update bars (helper name or style.width / className=bar)
	if !strings.Contains(html, "applyQuotaToRow") {
		t.Fatal("missing applyQuotaToRow")
	}
	hasBarUpdate := strings.Contains(html, "quotaBarClass") ||
		strings.Contains(html, "renderQuotaBar") ||
		strings.Contains(html, "updateQuotaBar") ||
		strings.Contains(html, "setQuotaBar") ||
		(strings.Contains(html, "className") && strings.Contains(html, "'bar")) ||
		(strings.Contains(html, "className") && strings.Contains(html, `"bar`)) ||
		strings.Contains(html, "data-bar-width") ||
		strings.Contains(html, ".bar > span") ||
		strings.Contains(html, "querySelector('.bar')") ||
		strings.Contains(html, `querySelector(".bar")`)
	if !hasBarUpdate {
		t.Fatal("enrich JS should update bars or data attributes")
	}
}

func TestRenderStatusPageStatusPillOffNeutral(t *testing.T) {
	htmlOff := RenderStatusPage(StatusResponse{Enabled: false, Version: "0.1.5"}, LangZhHant)
	if !strings.Contains(htmlOff, `status-pill warn`) && !strings.Contains(htmlOff, `status-pill off`) && !strings.Contains(htmlOff, `status-pill neutral`) {
		t.Fatal("when schedule off, status-pill must use neutral/warn/off class")
	}
}

func TestRenderStatusPageAbnormalFilterCount(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.5", Model: "gpt-5.4",
		Timezone: "Asia/Taipei", Times: []string{"21:00"},
		Accounts: []runstate.AccountView{
			{AuthIndex: "1", Name: "alice", Selected: true, Status: "success"},
			{AuthIndex: "2", Name: "bob", Selected: false, Status: "limited"},
			{AuthIndex: "3", Name: "old", Selected: false, Status: "unavailable"},
		},
	}, LangZhHant)
	if !strings.Contains(html, "異常 2") && !strings.Contains(html, "異常2") {
		t.Fatalf("abnormal filter tab should show count; want 異常 2")
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

func TestEnrichBarWidthOnBarNotParent(t *testing.T) {
	html := RenderStatusPage(StatusResponse{Enabled: true}, LangZhHant)
	if !strings.Contains(html, "bar.setAttribute('data-bar-width'") && !strings.Contains(html, `bar.setAttribute("data-bar-width"`) {
		t.Fatal("enrich must set data-bar-width on .bar (SSR consistency)")
	}
	if strings.Contains(html, "parent.setAttribute('data-bar-width'") || strings.Contains(html, `parent.setAttribute("data-bar-width"`) {
		t.Fatal("enrich must not set data-bar-width on parent hide-sm cell")
	}
	if strings.Contains(html, "剩/left/残") {
		t.Fatal("fmtQuotaWindow must not contain dead leftover locale push")
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
