# Run-now polling UX — implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:test-driven-development. Steps use checkbox (`- [ ]`) syntax. **Do not start TDD until the user explicitly confirms this plan in chat.**

**Goal:** Management UI polls `GET …/plugins/codex-selective-ping/status` for `running`, shows 執行中, locks Run now, and reloads only when the run finishes so history appears without guessing.

**Architecture:** Server-render `data-running` (+ disable Run buttons when already running). Replace fixed `setTimeout(…, 1500)` reload with JS: enter running mode → poll status JSON every 2.5s with Bearer key → `running === false` → `location.reload()`. 202 and 409 both enter the same mode. No runner/history persistence changes.

**Tech stack:** Go string-built HTML/JS in `internal/management`, existing `ui_test.go` / `i18n` tests, Docker Compose `dev` or `test` profile.

**Spec:** `docs/superpowers/specs/2026-09-22-run-now-polling-ux-design.md`

## Global constraints

- No Cloud Agent
- Chat progress in **正體中文**
- **No `git commit` / push / PR** until end-of-flow user confirmation
- Do not change `State.End()` history timing, retry interval, or run timeout
- Keep status JSON contract (`running` bool) as-is
- i18n: zh-Hant / en / ja for any new user-visible strings

## File map

| Path | Role |
|------|------|
| `internal/management/ui.go` | Markup bootstrap (`data-running`), disable attrs, JS poll / `enterRunningMode` / `runNow` |
| `internal/management/i18n.go` | New keys: `run_polling`, `run_already` (and wire `running_label` if unused) |
| `internal/management/ui_test.go` | HTML/JS assertion tests |
| `internal/management/i18n_test.go` | Key presence for 3 langs (if such test exists / extend) |
| READMEs | Optional one-liner that Run now waits until finish then refreshes — only if docs already describe Run now UX |

---

### Task 1: Failing UI tests — bootstrap + no fixed 1.5s reload

**Files:**
- Modify: `internal/management/ui_test.go`
- (Implementation later) `internal/management/ui.go`

- [ ] **Step 1: Write failing tests**

```go
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
	// Also reject the current production pattern if still present:
	if strings.Contains(html, "setTimeout(()=>location.reload(),1500)") || strings.Contains(html, "1500)}") && strings.Contains(html, "location.reload") && strings.Contains(html, "runNow") {
		// precise: the known current line
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
```

(Adjust exact helper/snippet names to match existing test style in `ui_test.go`; keep assertions on substrings as above.)

- [ ] **Step 2: RED in Docker**

```bash
docker compose up -d dev
docker compose exec -T dev go test ./internal/management/ -count=1 -run 'TestRenderStatusPageRunningBootstrap|TestRenderStatusPageRunNowPollsStatusNotFixedReload'
```

Expected: FAIL (missing `data-running` / still has 1500ms reload / missing poll markers).

- [ ] **Step 3: Minimal `ui.go` markup** so bootstrap test can pass:
  - On root element (e.g. `<body …>` or main shell): `data-running="%s"` with `true`/`false` from `st.Running`
  - Add `data-run-now` to both Run buttons (rail ~line 639 and empty-history ~line 708)
  - When `st.Running`: add `disabled` on those buttons; show a small badge/span with `data-i18n="running_label"` (and current-lang text)
  - Do **not** yet remove 1500ms or add poll JS if you prefer Task 2 for JS — but Task 1 GREEN may require at least `data-running` + disabled + label. Prefer split: Task 1 GREEN = bootstrap only; keep poll test in Task 2 if needed.

**Preferred split if RED is noisy:** keep both tests in Task 1 but implement markup first, leave poll test failing into Task 2. Or implement both markup+JS in one GREEN after both tests exist (still TDD: tests first, then code).

- [ ] **Step 4: GREEN for bootstrap test** (poll test may still fail until Task 2)

---

### Task 2: JS — `enterRunningMode` + poll + `runNow` rewrite

**Files:**
- Modify: `internal/management/ui.go` (script section ~`runNow`)
- Modify: `internal/management/ui_test.go` (ensure Task 1 poll test is GREEN)

- [ ] **Step 1: Confirm poll test still RED** (if not already written in Task 1, add it now and RED)

- [ ] **Step 2: Implement JS** (embedded in `ui.go`), behavior locked by spec:

```javascript
const POLL_MS = 2500;
const STATUS_URL = '/v0/management/plugins/codex-selective-ping/status';
const RUN_URL = '/v0/management/plugins/codex-selective-ping/run';
let pollTimer = null;

function setRunButtonsDisabled(on){
  document.querySelectorAll('[data-run-now]').forEach(function(b){ b.disabled = !!on; });
}
function showRunningBadge(on){
  const el = document.getElementById('running-badge');
  if(el){ el.hidden = !on; }
}
function enterRunningMode(msg){
  setRunButtonsDisabled(true);
  showRunningBadge(true);
  const o = document.getElementById('result');
  if(o && msg){ o.textContent = msg; }
  startPolling();
}
async function pollOnce(){
  const k = key();
  if(!k){ const o=document.getElementById('result'); if(o) o.textContent=msgNeedKey; return; }
  try{
    const r = await fetch(STATUS_URL,{headers:{'Authorization':'Bearer '+k}});
    const j = await r.json();
    if(j && j.running === false){
      if(pollTimer){ clearInterval(pollTimer); pollTimer=null; }
      location.reload();
      return;
    }
  }catch(e){
    const o=document.getElementById('result');
    if(o){ o.textContent=String(e); }
  }
}
function startPolling(){
  if(pollTimer) return;
  pollTimer = setInterval(pollOnce, POLL_MS);
  pollOnce();
}
async function runNow(){
  const o=document.getElementById('result'); const k=key();
  if(!k){ o.textContent=msgNeedKey; return; }
  o.textContent=msgStarting;
  try{
    const r=await fetch(RUN_URL,{method:'POST',headers:{'Authorization':'Bearer '+k}});
    const text=await r.text();
    o.textContent=text;
    // 202 accepted OR 409 already running → same UX
    if(r.ok || r.status===409){
      enterRunningMode(msgRunPolling || text);
    }
  }catch(e){ o.textContent=String(e); }
}
// bootstrap
(function(){
  const root = document.body;
  if(root && root.getAttribute('data-running')==='true'){
    enterRunningMode(msgRunPolling);
  }
})();
```

Wire `msgRunPolling` / `msgRunAlready` as `const` from Go `t("run_polling")` etc. (same pattern as `msgStarting`).

Remove any `setTimeout(()=>location.reload(),1500)` after run.

- [ ] **Step 3: GREEN**

```bash
docker compose exec -T dev go test ./internal/management/ -count=1 -run 'TestRenderStatusPageRunningBootstrap|TestRenderStatusPageRunNowPollsStatusNotFixedReload'
```

Expected: PASS.

---

### Task 3: i18n keys (zh-Hant / en / ja)

**Files:**
- Modify: `internal/management/i18n.go`
- Modify: `internal/management/i18n_test.go` (if key-list test exists)

- [ ] **Step 1: Failing test** that each lang map contains `run_polling` and `run_already` (and `running_label` still present)

Suggested copy:

| key | zh-Hant | en | ja |
|-----|---------|----|----|
| `run_polling` | 執行中，完成後會自動更新… | Running — page will refresh when finished… | 実行中。完了後に自動更新します… |
| `run_already` | 已有執行在進行，改為等待完成… | A run is already in progress — waiting for it to finish… | すでに実行中です。完了を待ちます… |

- [ ] **Step 2: RED → add keys → GREEN**

- [ ] **Step 3: Ensure `runNow` uses `msgRunAlready` when `r.status===409`** (small JS tweak if not done in Task 2)

---

### Task 4: Full suite + optional README touch + review gate

- [ ] **Step 1:**

```bash
docker compose exec -T dev go test ./... -count=1
```

Expected: all PASS.

- [ ] **Step 2 (optional):** If README Run-now section implies instant history, add one sentence: UI waits until the run finishes then refreshes (en / zh-Hant / ja). Skip if docs already accurate.

- [ ] **Step 3:** Code review (Critical / Important / Minor) in chat, **正體中文** summary of behavior + how to verify.

- [ ] **Step 4:** Ask user before any `git commit` / PR / release.

## Non-negotiables

- No TDD/implementation before **user confirms this plan**
- No Cloud Agent
- No commit until user confirms after summary
- History still written only on run `End()`

## Spec coverage checklist

| Spec item | Task |
|-----------|------|
| Poll status JSON `running` | Task 2 |
| Interval ~2.5s | Task 2 (`2500`) |
| Reload only when `running === false` | Task 2 |
| Remove fixed 1.5s reload | Task 1–2 |
| Bootstrap on `data-running` | Task 1 |
| Disable Run now while running | Task 1–2 |
| 202 and 409 → running mode | Task 2–3 |
| i18n zh/en/ja | Task 3 |
| No backend contract / history timing change | (none — do not touch runner/runstate persist) |
