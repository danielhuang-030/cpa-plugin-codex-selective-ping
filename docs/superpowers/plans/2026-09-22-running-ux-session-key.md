# Running UX + sessionStorage key — implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:test-driven-development. Steps use checkbox (`- [ ]`) syntax. **Do not start TDD until the user explicitly confirms this plan in chat.**

**Goal:** Clearer running UI + same-tab Management Key via sessionStorage so reload/polling does not false-need-key (#15).

**Architecture:** Management UI-only changes in `internal/management` (HTML/CSS/JS + i18n + string tests). No runner/history changes.

**Tech stack:** Go HTML/JS, Docker Compose `dev` tests.

**Spec:** `docs/superpowers/specs/2026-09-22-running-ux-session-key-design.md`  
**Issue:** #15

## Global constraints

- No Cloud Agent
- Chat in **正體中文**
- **No git commit** until end-of-flow user confirmation
- Still forbid `localStorage.setItem` for Management Key
- sessionStorage key: `codex-selective-ping:management-key`

---

### Task 1: Failing tests

**Files:** `internal/management/ui_test.go` (and i18n_test if needed)

- [ ] Assert HTML/JS contains `sessionStorage.setItem` / `sessionStorage.getItem` with `codex-selective-ping:management-key`
- [ ] Assert still **no** `localStorage.setItem` used for the management key (existing check may stay as “no localStorage.setItem” globally in page, or narrowed — keep current strict check if page already only uses localStorage for language)
- [ ] Assert running banner marker exists, e.g. `id="running-banner"` / `data-testid="running-banner"`
- [ ] When `Running: true`, banner not `hidden`; when false, banner hidden
- [ ] RED in Docker: `docker compose exec -T dev go test ./internal/management/ -count=1 -run 'SessionKey|RunningBanner'`

### Task 2: sessionStorage key helpers (GREEN)

**Files:** `internal/management/ui.go`

- [ ] `KEY_STORAGE = 'codex-selective-ping:management-key'`
- [ ] restore on load; persist when non-empty in `key()` path / save / run / poll
- [ ] Bootstrap running mode uses restored key so poll does not immediately show need_key

### Task 3: Running banner UI (GREEN)

**Files:** `internal/management/ui.go`, `i18n.go` if new copy

- [ ] Add `#running-banner` near rail status (chip stays)
- [ ] CSS: noticeable warn styles
- [ ] `showRunningBadge` → also toggle banner (rename or add `setRunningUI(on)`)
- [ ] i18n zh-Hant / en / ja for any new strings

### Task 4: Full suite + review gate

- [ ] `docker compose exec -T dev go test ./... -count=1`
- [ ] Code review + 正體中文 summary
- [ ] Ask before commit / PR / bump / release (issue flow: branch → PR → merge → close #15 → bump → release)

## Non-negotiables

- No TDD before **user confirms this plan**
- No commit until user confirms after summary
