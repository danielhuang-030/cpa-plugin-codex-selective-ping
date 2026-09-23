# Model Row Layout (Scheme 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Keep「更新模型」on the same row as the「模型」label, with `#model-select` full-width below, so long model ids never wrap the button.

**Architecture:** Taste redesign–preserve of one management rail metric. Stack markup into `metric-model-top` + select; CSS column layout using existing tokens only. Behavior (`loadModels` / `X-Csp-Origin`) unchanged.

**Tech Stack:** Go string HTML/CSS in `internal/management/ui.go`; Go tests in `ui_test.go`; Docker `go test`.

**Spec:** `docs/superpowers/specs/2026-09-23-model-row-layout-design.md`

## Global Constraints

- Preserve `#model-select`, `#refresh-models`, `data-testid`, i18n keys, `btn-secondary`.
- Do not change non-model `.metric` rows or page theme tokens.
- No version bump/commit/release unless the user asks after green tests.
- Docker tests need `sudo` and often `PATH=/usr/local/go/bin:...` on this box; do not apt-get.

---

## Task 1: Failing UI layout tests

**Files:**
- Modify: `internal/management/ui_test.go`

- [x] **Step 1: Write failing test** asserting rendered HTML contains `metric-model-top`, and within the model metric the substring order is `refresh-models` before `id="model-select"` (button above select). Also assert CSS has no `flex-wrap: wrap` on `.metric.metric-model` (or assert `metric-model-top` + `#model-select { width: 100%`).

- [x] **Step 2: Run**  
  `sudo docker compose exec -T -e PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin dev go test ./internal/management/ -run 'Test.*Model|Test.*Refresh|TestRender' -count=1`  
  Expect RED for the new assertions.

---

## Task 2: Implement markup + CSS

**Files:**
- Modify: `internal/management/ui.go`

- [x] **Step 1: Update HTML** for the model metric to scheme 1 structure (`metric-model-top` wrapping label + button; select sibling below).

- [x] **Step 2: Update CSS**  
  - `.metric.metric-model`: column, stretch, gap 8px; remove wrap.  
  - `.metric-model-top`: space-between row, nowrap button.  
  - `#model-select`: `width:100%; min-width:0`; remove `max-width:58%`.

- [x] **Step 3: Run** same test filter → GREEN. Then full:  
  `sudo docker compose exec -T -e PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin dev go test ./... -count=1`

---

## Task 3: Smoke notes

- [x] Confirm mock still matches: `docs/superpowers/mocks/model-row-layout-mock.html` middle card.
- [x] Report summary in Traditional Chinese; do not commit unless asked.

