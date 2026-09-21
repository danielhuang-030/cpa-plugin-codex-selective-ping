# Management UI Taste Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the CPA Management page for Codex Selective Ping to match the locked v3 mock (warm editorial ops console) without changing ping/whitelist/schedule behavior.

**Architecture:** Keep server-rendered HTML from `RenderStatusPage` in `internal/management/ui.go`. Replace layout/CSS/markup landmarks to match `docs/superpowers/mocks/management-ui-mock-v3.html`. Preserve existing element IDs and JS (`schedule_enabled`, `tz`, `management-key`, `saveCfg`, `runNow`, `addTime`, `setAll`, theme sync, quota enrich). Extend i18n keys in `internal/management/i18n.go`. Prove structure with string/assert tests in `ui_test.go`; run via Docker Compose `test` service.

**Tech Stack:** Go, embedded HTML/CSS/JS, existing management package tests, Docker Compose (`compose.yaml` profile `tools`).

## Global Constraints

- Spec: `docs/superpowers/specs/2026-09-21-management-ui-taste-redesign.md`
- Mock source of truth: `docs/superpowers/mocks/management-ui-mock-v3.html`
- Path A only: edit `ui.go` / `i18n.go` / tests — no HTML file extract
- Preserve behavior: save body uses `schedule_enabled` (not host `enabled`); run-now uses saved whitelist; last-run empty vs filled; theme follows CPA `data-theme`
- No Cloud Agent; tests: `docker compose --profile tools run --rm test` (or equivalent profile)
- sp-flow: TDD — failing test before production markup changes for each task
- Do not bump version/release until UI tasks green and summarized (optional final task)

---

### Task 1: Landmark tests for v3 shell (RED → GREEN)

**Files:**
- Modify: `internal/management/ui_test.go`
- Modify: `internal/management/ui.go` (minimal landmarks only after RED)
- Test: `internal/management/ui_test.go`

**Interfaces:**
- Consumes: `RenderStatusPage(st StatusResponse, lang Lang) string`
- Produces: HTML containing `class="shell"`, `class="rail"`, `class="workspace"`

- [x] **Step 1: Write the failing test**

```go
func TestRenderStatusPageV3ShellLandmarks(t *testing.T) {
	html := RenderStatusPage(StatusResponse{Version: "0.1.5"}, LangZhHant)
	for _, needle := range []string{`class="shell"`, `class="rail"`, `class="workspace"`} {
		if !strings.Contains(html, needle) {
			t.Fatalf("missing v3 landmark %s", needle)
		}
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `docker compose --profile tools run --rm test ./internal/management/ -count=1 -run TestRenderStatusPageV3ShellLandmarks`

Expected: FAIL — missing `class="shell"` (or rail/workspace)

- [x] **Step 3: Write minimal implementation**

Wrap existing page body in:

```html
<div class="shell"><aside class="rail">…</aside><div class="workspace">…</div></div>
```

Move brand/status into rail; keep old sections temporarily inside workspace if needed — enough to pass landmark asserts.

- [x] **Step 4: Run test to verify it passes**

Run: `docker compose --profile tools run --rm test ./internal/management/ -count=1 -run TestRenderStatusPageV3ShellLandmarks`

Expected: PASS

- [x] **Step 5: Commit**

```bash
git add internal/management/ui.go internal/management/ui_test.go
git commit -m "test+feat(ui): add v3 shell/rail/workspace landmarks"
```

---

### Task 2: Rhythm schedule grid + next slot (TDD)

**Files:**
- Modify: `internal/management/ui.go`
- Modify: `internal/management/ui_test.go`
- Modify: `internal/management/i18n.go` (keys: `rhythm_title`, `slot_next`, `slot_past` as needed)

**Interfaces:**
- Consumes: `st.Times []string`, `st.Timezone`, `st.Enabled`, `st.NextRun` (`interface{}` on `StatusResponse`)
- Produces: markup with `class="timeline"`, each time as `class="slot"`, upcoming marked `class="slot next"`; keep `#schedule_enabled`, `#tz`, `#new-time`, `addTime()`

- [x] **Step 1: Write the failing test**

```go
func TestRenderStatusPageRhythmTimeline(t *testing.T) {
	html := RenderStatusPage(StatusResponse{
		Timezone: "Asia/Taipei",
		Enabled:  true,
		Times:    []string{"06:00", "11:00", "16:00", "21:00"},
		NextRun:  "21:00", // use actual field name from StatusResponse — adjust to real field
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
```

(Adjust `NextRun` to the real `StatusResponse` field used today for “下次執行”.)

- [x] **Step 2: Run — expect FAIL** (`docker compose --profile tools run --rm test ./internal/management/ -count=1 -run TestRenderStatusPageRhythmTimeline`)

- [x] **Step 3: Implement timeline rendering** matching mock: replace chip row with slot grid; compute next slot from status; keep add-time controls

- [x] **Step 4: Run — expect PASS** (+ re-run Task 1 test)

- [x] **Step 5: Commit** `feat(ui): render schedule as rhythm timeline with next highlight`

---

### Task 3: Account cards + empty whitelist panel (TDD)

**Files:**
- Modify: `internal/management/ui.go`
- Modify: `internal/management/ui_test.go`
- Modify: `internal/management/i18n.go` (keys: `accounts_empty_title`, `accounts_empty_body`, filter labels if shown)

**Interfaces:**
- Consumes: `st.Accounts` / allowlist selection fields already used by checkboxes (`preferID` / auth index)
- Produces: `class="account-grid"`, each `class="acct"` (selected adds `selected`); when zero selected, show `id="sec-accounts-empty"` empty panel; keep checkbox values + `setAll`

- [x] **Step 1: Failing tests**

```go
func TestRenderStatusPageAccountCards(t *testing.T) {
	html := RenderStatusPage(/* fixture with 2 accounts, 1 selected */, LangZhHant)
	if !strings.Contains(html, `class="account-grid"`) {
		t.Fatal("missing account-grid")
	}
	if !strings.Contains(html, `class="acct`) {
		t.Fatal("missing acct card")
	}
}

func TestRenderStatusPageAccountsEmptyState(t *testing.T) {
	html := RenderStatusPage(/* no selected accounts */, LangZhHant)
	if !strings.Contains(html, `id="sec-accounts-empty"`) && !strings.Contains(html, `data-i18n="accounts_empty_title"`) {
		t.Fatal("missing empty whitelist empty-state")
	}
}
```

- [x] **Step 2: Run — FAIL**

- [x] **Step 3: Replace `<table>` accounts block with card list + empty state; preserve `name`/value on checkboxes used by `selectedAccounts()`

- [x] **Step 4: PASS** + existing checkbox/auth-index tests still green

- [x] **Step 5: Commit** `feat(ui): account cards and empty whitelist state`

---

### Task 4: Rail actions + CTA order (TDD)

**Files:**
- Modify: `internal/management/ui.go`
- Modify: `internal/management/ui_test.go`

**Interfaces:**
- Produces: `#management-key`, `onclick="saveCfg()"`, `onclick="runNow()"` inside `.rail` (or `.primary-stack`); primary save appears before run-now in HTML order

- [x] **Step 1: Failing test**

```go
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
	if rail < 0 || key < rail {
		t.Fatal("actions should live in rail")
	}
}
```

- [x] **Step 2–4:** RED → move actions into rail with v3 button classes → GREEN; keep `TestRenderStatusPageSaveUsesScheduleEnabledNotEnabled` passing

- [x] **Step 5: Commit** `feat(ui): move save/run/refresh into left rail with CTA order`

---

### Task 5: Receipt last-run + empty last-run (TDD)

**Files:**
- Modify: `internal/management/ui.go`
- Modify: `internal/management/ui_test.go`
- Modify: `internal/management/i18n.go` (`last_run_empty_title`, `last_run_empty_body`)

**Interfaces:**
- Consumes: existing last-run summary on `StatusResponse`
- Produces: `class="run"` with `run-summary` when present; empty state when absent (already `no_last_run` — keep key or add new)

- [x] **Step 1: Failing tests** for `class="run-summary"` when last run set, and empty panel when nil

- [x] **Step 2–4:** RED → implement receipt layout → GREEN

- [x] **Step 5: Commit** `feat(ui): receipt-style last run and empty state`

---

### Task 6: Warm v3 CSS tokens + dark theme parity (TDD smoke)

**Files:**
- Modify: `internal/management/ui.go` (`<style>` block)
- Modify: `internal/management/ui_test.go`

- [x] **Step 1: Failing test** asserting CSS contains warm token markers, e.g. `--accent:` and `data-theme="dark"` rules (or `:root[data-theme="dark"]`), and theme sync JS still present (`TestRenderStatusPageThemeSyncJS`)

- [x] **Step 2–4:** Port mock CSS into embedded stylesheet; map CPA theme attribute the page already uses; GREEN

- [x] **Step 5: Commit** `feat(ui): apply v3 warm editorial CSS tokens`

---

### Task 7: i18n completeness for new strings

**Files:**
- Modify: `internal/management/i18n.go`
- Modify: `internal/management/i18n_test.go` (or add if missing)

- [x] **Step 1:** Test that every new `data-i18n` key used in `ui.go` exists for `zh` / `en` / `ja` maps

- [x] **Step 2–4:** RED → fill translations → GREEN

- [x] **Step 5: Commit** `feat(i18n): strings for v3 management UI`

---

### Task 8: Full package regression in Docker + issue comment

**Files:** none (verify)

- [x] **Step 1:** `docker compose --profile tools run --rm test` (full `./...`)

Expected: all packages ok

- [x] **Step 2:** Manually spot-check rendered HTML fixture against mock landmarks (optional local file write under `/tmp`)

- [x] **Step 3:** Comment on GitHub issue #8 with commit SHAs + note behavior unchanged

- [x] **Step 4:** Commit only if docs/README screenshots need update; otherwise stop for human review before version bump

---

## Plan self-review

1. **Spec coverage:** rail, rhythm, principles panel, account cards, receipt, empty states, path A, Docker TDD, no behavior change → Tasks 1–7  
2. **Placeholders:** NextRun field name must be resolved against `StatusResponse` at Task 2 start (read `handler.go`) — not left TBD in code  
3. **IDs:** `schedule_enabled`, `tz`, `management-key`, `saveCfg`, `runNow` preserved across tasks  

## Execution

Preferred: subagent-driven-development on box worktree / main, Docker for every `go test`. No Cloud Agent.
