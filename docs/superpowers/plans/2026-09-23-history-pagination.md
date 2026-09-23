# History list frontend pagination — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.  
> **Do not start TDD until the user confirms this plan in chat.**  
> **sp-flow:** No `git commit` / push / PR / release until end-of-flow user confirmation. Chat in 正體中文. No Cloud Agent. Tests via Docker Compose `dev`.

**Goal:** Paginate the Management UI run-history list on the frontend (default 10 rows/page; UI select 10/20/50; remember size in `localStorage`), without changing plugin config or the status API.

**Architecture:** Approach A — server still renders all `.hist-item` rows from `run_history`; JS shows/hides the current page. Pager chrome sits below `.hist`. Page size key `csp-hist-page-size`; invalid/missing → 10; changing size resets to page 1.

**Tech Stack:** Go management UI (`internal/management` HTML/CSS/JS/i18n), Docker Compose `dev` for `go test`.

**Spec:** `docs/superpowers/specs/2026-09-23-history-pagination-design.md`

## Global Constraints

- Chat / progress: 正體中文
- Tests: `docker compose exec -T dev go test ./internal/management/ -count=1` (or `./...`)
- No Cloud Agent
- **No git commit / push / PR / release until user confirms after review**
- No new plugin config fields; do not change `history_limit` or status `run_history` shape
- localStorage key exactly `csp-hist-page-size`; allowed sizes exactly `10`, `20`, `50`; default `10`
- Empty history: no pager

## File map

| File | Role |
| --- | --- |
| `internal/management/ui.go` | Pager HTML after history list; CSS; JS `initHistPager` + show/hide |
| `internal/management/i18n.go` | zh-Hant / en / ja strings for pager |
| `internal/management/ui_test.go` | Assert pager markers, options, JS hooks |
| `internal/management/i18n_test.go` | Required keys list |
| READMEs (en / zh-Hant / ja) | One-line note: UI paginates history; size in browser only |

---

### Task 1: i18n keys for pager

**Files:**
- Modify: `internal/management/i18n.go`
- Modify: `internal/management/i18n_test.go`

**Produces:** Keys (all three langs):
- `hist_pager_prev` — 上一頁 / Previous / 前へ
- `hist_pager_next` — 下一頁 / Next / 次へ
- `hist_pager_page` — 第 %d／%d 頁 (en: Page %d / %d; ja: %d／%d ページ) — **or** split into static label + JS fills numbers: prefer **`hist_pager_page`** as format with two `%d`, applied in JS via a data attribute or build label in JS from `data-i18n` stubs. Simpler for string tests: static keys `hist_pager_prev`, `hist_pager_next`, `hist_pager_per_page` ("每頁" / "Per page" / "件数"), and page label built in JS as `` `${n} / ${N}` `` without i18n format — **locked:** use static keys only; page indicator is numeric `n / N` in JS (no translated "Page" required beyond optional `hist_pager_of` if needed). Minimum keys: `hist_pager_prev`, `hist_pager_next`, `hist_pager_per_page`.

- [ ] **Step 1: Failing test** — add the three keys to the required-keys slice in `i18n_test.go` so missing translations fail.

- [ ] **Step 2: RED** — `docker compose exec -T dev go test ./internal/management/ -run TestI18n -count=1` (or the existing required-keys test name).

- [ ] **Step 3: Implement** — add zh-Hant / en / ja entries for `hist_pager_prev`, `hist_pager_next`, `hist_pager_per_page`.

- [ ] **Step 4: GREEN** — same test command.

- [ ] **Step 5:** Do **not** commit.

---

### Task 2: Pager HTML + CSS markers (TDD)

**Files:**
- Modify: `internal/management/ui.go` (`renderRunHistoryList` and CSS block)
- Modify: `internal/management/ui_test.go`

**Produces:** When `len(runs) > 0`, after the closing `</div>` of `.hist`, emit pager:

```html
<div class="hist-pager" data-testid="hist-pager">
  <button type="button" data-hist-prev data-i18n="hist_pager_prev">…</button>
  <span data-hist-page-label>1 / 1</span>
  <button type="button" data-hist-next data-i18n="hist_pager_next">…</button>
  <label><span data-i18n="hist_pager_per_page">…</span>
    <select data-hist-page-size>
      <option value="10">10</option>
      <option value="20">20</option>
      <option value="50">50</option>
    </select>
  </label>
</div>
```

Use `t("hist_pager_*")` for button/label text. CSS: flex row, gap, align with existing warm ops-console tokens; disabled buttons look muted.

- [ ] **Step 1: Failing test** in `ui_test.go` (extend a fixture that already has ≥1 history run, e.g. `TestRenderStatusPageRunHistoryList` or new `TestRenderStatusPageHistPager`):

```go
func TestRenderStatusPageHistPager(t *testing.T) {
	// Build StatusResponse with ≥2 RunHistory entries (reuse helpers from existing history tests).
	htmlOut := renderStatusPage(/* … */) // same pattern as TestRenderStatusPageRunHistoryList
	if !strings.Contains(htmlOut, `data-testid="hist-pager"`) {
		t.Fatal("missing hist-pager")
	}
	for _, v := range []string{`value="10"`, `value="20"`, `value="50"`} {
		if !strings.Contains(htmlOut, v) {
			t.Fatalf("missing page size option %s", v)
		}
	}
	if !strings.Contains(htmlOut, `data-hist-prev`) || !strings.Contains(htmlOut, `data-hist-next`) {
		t.Fatal("missing pager prev/next hooks")
	}
	if !strings.Contains(htmlOut, `data-i18n="hist_pager_prev"`) {
		t.Fatal("missing hist_pager_prev i18n")
	}
}

func TestRenderStatusPageHistPagerAbsentWhenEmpty(t *testing.T) {
	// Status with empty RunHistory and nil LastRun → no data-testid="hist-pager"
}
```

- [ ] **Step 2: RED** — `docker compose exec -T dev go test ./internal/management/ -run 'TestRenderStatusPageHistPager' -count=1`

- [ ] **Step 3: Implement** HTML + minimal CSS in `ui.go`.

- [ ] **Step 4: GREEN** — same filter; also ensure existing history tests still pass.

- [ ] **Step 5:** Do **not** commit.

---

### Task 3: JS pagination logic (TDD via string assertions)

**Files:**
- Modify: `internal/management/ui.go` (script section)
- Modify: `internal/management/ui_test.go`

**Produces:** JS (concept):

```javascript
const HIST_PAGE_SIZE_KEY = 'csp-hist-page-size';
const HIST_PAGE_SIZES = [10, 20, 50];
const HIST_PAGE_SIZE_DEFAULT = 10;
let histPage = 1;

function readHistPageSize(){
  try {
    const n = parseInt(localStorage.getItem(HIST_PAGE_SIZE_KEY) || '', 10);
    if (HIST_PAGE_SIZES.indexOf(n) >= 0) return n;
  } catch (e) {}
  return HIST_PAGE_SIZE_DEFAULT;
}
function writeHistPageSize(n){
  try { localStorage.setItem(HIST_PAGE_SIZE_KEY, String(n)); } catch (e) {}
}
function applyHistPage(){
  const root = document.querySelector('[data-testid="run-history"]');
  const pager = document.querySelector('[data-testid="hist-pager"]');
  if (!root || !pager) return;
  const items = Array.prototype.slice.call(root.querySelectorAll('.hist-item'));
  const sizeSel = pager.querySelector('[data-hist-page-size]');
  let size = readHistPageSize();
  if (sizeSel) {
    sizeSel.value = String(size);
    size = parseInt(sizeSel.value, 10) || HIST_PAGE_SIZE_DEFAULT;
  }
  const total = items.length;
  const pages = Math.max(1, Math.ceil(total / size));
  if (histPage > pages) histPage = pages;
  if (histPage < 1) histPage = 1;
  const start = (histPage - 1) * size;
  items.forEach(function(el, i){
    el.style.display = (i >= start && i < start + size) ? '' : 'none';
  });
  const label = pager.querySelector('[data-hist-page-label]');
  if (label) label.textContent = histPage + ' / ' + pages;
  const prev = pager.querySelector('[data-hist-prev]');
  const next = pager.querySelector('[data-hist-next]');
  if (prev) prev.disabled = histPage <= 1;
  if (next) next.disabled = histPage >= pages;
}
function initHistPager(){
  const pager = document.querySelector('[data-testid="hist-pager"]');
  if (!pager) return;
  const sizeSel = pager.querySelector('[data-hist-page-size]');
  if (sizeSel) {
    sizeSel.value = String(readHistPageSize());
    sizeSel.addEventListener('change', function(){
      const n = parseInt(sizeSel.value, 10);
      writeHistPageSize(n);
      histPage = 1;
      applyHistPage();
    });
  }
  const prev = pager.querySelector('[data-hist-prev]');
  const next = pager.querySelector('[data-hist-next]');
  if (prev) prev.addEventListener('click', function(){ histPage--; applyHistPage(); });
  if (next) next.addEventListener('click', function(){ histPage++; applyHistPage(); });
  applyHistPage();
}
// Call initHistPager() near other bootstrap (end of script or DOMContentLoaded).
```

- [ ] **Step 1: Failing test**

```go
func TestRenderStatusPageHistPagerJS(t *testing.T) {
	htmlOut := /* render with history */
	for _, needle := range []string{
		`csp-hist-page-size`,
		`HIST_PAGE_SIZE_DEFAULT = 10`, // or `const HIST_PAGE_SIZE_DEFAULT=10` — match exact emit
		`initHistPager`,
		`histPage = 1`, // reset on size change
	} {
		if !strings.Contains(htmlOut, needle) {
			t.Fatalf("missing JS needle %q", needle)
		}
	}
}
```

Adjust needles to match the exact JS you emit (keep key + `initHistPager` + default 10 + reset to page 1 on size change).

- [ ] **Step 2: RED**

- [ ] **Step 3: Implement** JS + call `initHistPager()` once on load (same place other inits run; if none, append call at end of script).

- [ ] **Step 4: GREEN** — `docker compose exec -T dev go test ./internal/management/ -run 'HistPager' -count=1`

- [ ] **Step 5:** Do **not** commit.

---

### Task 4: README note + full suite

**Files:**
- Modify: `README.md`, `README.zh-Hant.md` (and `README.ja.md` if present) — one sentence under Management UI / history: list is paginated in the browser (default 10; 10/20/50; preference in localStorage, not plugin config).

- [ ] **Step 1:** Add the README sentences.

- [ ] **Step 2: Full suite** — `docker compose exec -T dev go test ./... -count=1` → all green.

- [ ] **Step 3:** Do **not** commit.

---

### Task 5: Code review + chat summary (coordinator)

- [ ] Run requesting-code-review on dirty tree (no commit).
- [ ] Send 正體中文 summary: behavior, how verified, review findings, `git status`.
- [ ] Ask user whether to commit / open issue+PR / bump+release (完整出貨).

---

## Spec coverage checklist

| Spec item | Task |
| --- | --- |
| Frontend-only show/hide | 3 |
| Default 10; choices 10/20/50 | 2, 3 |
| localStorage `csp-hist-page-size` | 3 |
| No plugin config | (constraint; no task adds fields) |
| Pager below list; empty → no pager | 2 |
| Prev/next disabled at ends | 3 |
| Size change → page 1 | 3 |
| i18n zh/en/ja | 1 |
| Expand behavior unchanged | (no change to `toggleHist`) |
| Out of scope API/`history_limit` | (no tasks touch them) |

## Execution note

After user confirms this plan: open GitHub issue (title e.g. “History list UI pagination”), branch `feature/history-pagination`, then execute tasks with **subagent-driven-development** (or inline). Still **no commit** until Task 5 user approval.
