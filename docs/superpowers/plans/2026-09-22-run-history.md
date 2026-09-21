# Run history persistence — implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:test-driven-development. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist a newest-first run history in `run_history.json` with `history_limit` (default 60), migrate from `last_run.json` (then delete it), and show latest receipt + history list in the Management UI.

**Architecture:** Extend `internal/runstate` to hold `[]Summary`, load/save versioned JSON, trim on write. Config gains `history_limit`. Management status adds `run_history`; UI renders list under the receipt. All tests via Docker Compose `dev`/`test` service. **No git commit** until user confirms after sp-flow.

**Tech stack:** Go, existing `internal/runstate` + `internal/config` + `internal/management`, Docker Compose.

**Spec:** `docs/superpowers/specs/2026-09-22-run-history-design.md` · Issue #9

---

### Task 1: Config — `history_limit`

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `registration.go` (ConfigFields description)

- [ ] **Step 1: Failing test** — parse omit → 60; `history_limit: 10` → 10; `0`/`-1` → 60 (or clamp per design: invalid → 60, positive clamped ≥1 — implement as: ≤0 → 60)

- [ ] **Step 2: Run in Docker** — `docker compose exec -T dev go test ./internal/config/ -count=1` → RED

- [ ] **Step 3: Minimal implementation** — field on `Config`, JSON/YAML parse, default in `DefaultConfig`

- [ ] **Step 4: Registration** — add ConfigField `history_limit`

- [ ] **Step 5: GREEN** — re-run config tests

---

### Task 2: Persist format + migration

**Files:**
- Modify: `internal/runstate/persist.go`
- Modify: `internal/runstate/persist_test.go`
- Modify: `internal/runstate/runstate.go` (store slice; `LastRun` = first)

- [ ] **Step 1: Failing tests**
  - Write history file with 2 runs; reload → both present, `LastRun` = newest
  - Legacy single `Summary` at `last_run.json` beside target path → migrate to `run_history.json`, delete legacy
  - Append when over limit → trimmed to N, newest kept
  - `DefaultStatePath` / `ResolveStatePath` use `run_history.json`

- [ ] **Step 2: RED in Docker** — `docker compose exec -T dev go test ./internal/runstate/ -count=1`

- [ ] **Step 3: Implement** — `historyFile` struct `{Version int, Runs []Summary}`; load/migrate/save; `End` prepends + trim using limit from… **pass limit into State** (e.g. `SetHistoryLimit(n)` or save with limit arg)

- [ ] **Step 4: Wire limit** — plugin applies `cfg.HistoryLimit` when configuring persist

- [ ] **Step 5: GREEN**

---

### Task 3: Status API exposes `run_history`

**Files:**
- Modify: `internal/runstate/runstate.go` (`StatusSnapshot`)
- Modify: `internal/management/handler.go`
- Modify: tests under `internal/management/` / `internal/runstate/`

- [ ] **Step 1: Failing test** — snapshot/status JSON includes `run_history` array; `last_run` still newest

- [ ] **Step 2: RED → implement → GREEN**

---

### Task 4: Management UI history list + i18n

**Files:**
- Modify: `internal/management/ui.go`
- Modify: `internal/management/i18n.go`
- Modify: `internal/management/ui_test.go`, `i18n_test.go`

- [ ] **Step 1: Failing UI test** — with 2+ runs in status, HTML contains history list marker (e.g. `data-testid="run-history"` or `data-i18n="run_history"`) and more than one run row

- [ ] **Step 2: Implement** — receipt for `[0]`; compact list for all runs (or `[1:]` + highlight latest — prefer list all with latest also as receipt)

- [ ] **Step 3: i18n** zh-Hant / en / ja keys for section title/empty

- [ ] **Step 4: GREEN**

---

### Task 5: Docs + full test suite

**Files:**
- Modify: `README.md`, `README.zh-Hant.md`, `README.ja.md` (config table + persist path text)

- [ ] **Step 1: Update READMEs** — `history_limit`, `run_history.json`, migration note

- [ ] **Step 2: Full suite** — `docker compose exec -T dev go test ./... -count=1`

- [ ] **Step 3: Comment on issue #9** with progress (optional)

---

### Task 6: Code review + chat summary

- [ ] Run requesting-code-review checklist against working tree
- [ ] SendToUser summary; **ask before any git commit**

## Non-negotiables

- No Cloud Agent
- No `git commit` / push / PR / release until user confirms after summary
- Executors: working tree only
