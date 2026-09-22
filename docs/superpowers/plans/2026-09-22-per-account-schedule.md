# Per-account schedule + Management UI redesign — implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:test-driven-development. Steps use checkbox (`- [ ]`) syntax for tracking. **Do not start TDD until the user explicitly confirms this plan in chat.**

**Goal:** Per-account `times` via `account_times` (inherit global when missing/empty); schedule fires only matching accounts; full Management UI redo without 5h/weekly; richer expandable per-account history (manual/`force` included). Issue #11.

**Architecture:** Extend `config` with `account_times` + helpers (`EffectiveTimes`, `UnionTimes`, `AccountsForSlot`, prune-on-validate). Scheduler waits on **union** of effective times and passes the fire instant into `onFire`; plugin filters `cfg.Accounts` to that slot before `Runner.Run`. Management UI rebuilt around global rhythm + account cards (inherit/custom) + expandable history. Prefer JSON PATCH shape already used by `saveCfg()`.

**Tech stack:** Go 1.23, Docker Compose `dev` (`docker compose exec -T dev go test …`). **No git commit** until end-of-flow user confirmation (sp-flow). Feature branch + PR; delete branch after merge to `main`.

**Spec:** `docs/superpowers/specs/2026-09-22-per-account-schedule-design.md`

## Global constraints

- Chat: Traditional Chinese via SendToUser; code/paths/commands may stay English
- No Cloud Agent
- No TDD/implementation before **user confirms this plan**
- No `git commit` / push / PR / release until user confirms after summary
- Keep `accounts: []string`; do not change limited-retry semantics
- Mode strings stay as today: scheduled run `mode=scheduled`, manual `mode=force`
- Config JSON key: `account_times` (map[string][]string)

---

### Task 1: Config — `account_times` + helpers

**Files:**
- Modify: `internal/config/config.go`, `internal/config/config_test.go`
- Modify: `registration.go` (optional ConfigField note / description only if CPA supports map; otherwise document “UI / JSON only”)

**Interfaces (produce):**
- `Config.AccountTimes map[string][]string` with `json:"account_times,omitempty"`
- `func EffectiveTimes(cfg Config, account string) []string`
- `func UnionTimes(cfg Config) []string` — unique sorted HH:MM across allowlist effective times; if allowlist empty, return global `Times` (or empty — pick **return global Times** for next-run display stability when nobody selected)
- `func AccountsForSlot(cfg Config, hhmm string) []string` — allowlisted accounts whose effective times contain normalized `hhmm`
- Validate: each custom list uses same HH:MM normalization as `Times`; **empty custom list ⇒ delete key (inherit)**; **prune keys not in `Accounts`**

- [ ] **Step 1: Failing tests** in `config_test.go`:
  - JSON with `account_times` parses; omit → nil/empty map
  - `EffectiveTimes` inherit vs override
  - empty slice in map → inherit after Validate
  - invalid HH:MM in map → error
  - orphan key pruned when account not in `Accounts`
  - `UnionTimes` / `AccountsForSlot` examples from spec (alice global, bob custom)
- [ ] **Step 2: RED** — `docker compose exec -T dev go test ./internal/config/ -count=1`
- [ ] **Step 3: Implement** struct field, JSON parse, YAML subset for nested map (or document JSON-only if YAML subset stays too weak — prefer implement nested YAML: `account_times:` then `  email:` then `    - HH:MM`), Validate prune/normalize, helpers
- [ ] **Step 4: GREEN**
- [ ] **Step 5: Deep-copy `AccountTimes` in `plugin.Config()`** when touching plugin (can be Task 3)

---

### Task 2: Scheduler — union times + pass fire instant

**Files:**
- Modify: `internal/scheduler/scheduler.go`, `internal/scheduler/scheduler_test.go`

**Interfaces:**
- Change `onFire` to `func(context.Context, time.Time)` where `time.Time` is the scheduled fire instant in the schedule location
- `Start` uses `config.UnionTimes(cfg)` (not raw `cfg.Times`) when building the loop’s time list
- Keep `NextRun(now, loc, times []string)` as-is; add tests for union list behavior at call site or thin wrapper `NextRunFromConfig(now, cfg)` if clearer

- [ ] **Step 1: Failing tests**
  - Existing NextRun tests still pass
  - New: given cfg with bob-only extra slot, union includes that slot (test via exported helper or Start with fake clock if available — prefer unit-test `UnionTimes` in config + scheduler still calls it)
  - New: when fire callback invoked, second arg matches waited `next`
- [ ] **Step 2: RED**
- [ ] **Step 3: Implement** signature change + `UnionTimes` wiring; update all `New(...)` call sites
- [ ] **Step 4: GREEN**

---

### Task 3: Plugin + runner — slot-filtered scheduled runs

**Files:**
- Modify: `internal/plugin/plugin.go` (+ test if present / add `plugin_test.go`)
- Modify: `internal/runner/runner.go` only if needed (prefer **no runner API change**: copy cfg and set `Accounts` to `AccountsForSlot`)

- [ ] **Step 1: Failing test** — plugin/scheduler callback path (table-driven on helpers is OK if plugin hard to spin): `AccountsForSlot` at `06:00` returns only inheriting allowlist accounts; manual/`force` path still uses full allowlist
- [ ] **Step 2: Implement** in `New` callback:

```go
p.Sched = scheduler.New(func(ctx context.Context, at time.Time) {
    p.mu.Lock()
    cfg := p.cfg
    p.mu.Unlock()
    slot := at.Format("15:04")
    cfg.Accounts = config.AccountsForSlot(cfg, slot)
    p.Runner.Run(ctx, cfg, false)
}, p.State.SetNextRun)
```

- Deep-copy maps/slices in `Config()`
- Manual `StartManualRun` unchanged (full allowlist, `force=true`)
- [ ] **Step 3: GREEN** — `docker compose exec -T dev go test ./internal/plugin/ ./internal/runner/ ./internal/scheduler/ ./internal/config/ -count=1`

---

### Task 4: Management API status — expose `account_times`

**Files:**
- Modify: `internal/management/handler.go`, `handler_test.go`

- [ ] Add `AccountTimes map[string][]string `json:"account_times,omitempty"`` to `StatusResponse`
- [ ] Fill from `cfg.AccountTimes` in `status()`
- [ ] Test JSON contains `account_times` when set
- [ ] GREEN

Note: browser `saveCfg()` PATCHes CPA `/config` with JSON body; host reloads plugin config via existing decode — ensure `config.Parse` accepts `account_times` (Task 1). No new management write endpoint required unless CPA only forwards known fields (if so, confirm during impl; fall back to encoding map in a string field only if blocked — YAGNI unless proven).

---

### Task 5: Management UI — full redesign (no 5h/weekly; per-account times; history by account)

**Files:**
- Modify: `internal/management/ui.go`, `ui_test.go`, `i18n.go`, `i18n_test.go`
- Optional mock: `docs/superpowers/mocks/` only if helpful; not required to ship

**UI requirements (must):**
1. New IA: rail/actions + global rhythm + account-centric cards + run/history
2. Account cards: select checkbox; inherit vs custom toggle; custom times editor; **remove all 5h / weekly / `data-col="five_hour"` / `col_weekly` display and related client enrich that only feeds those columns** (dead JS can go)
3. `saveCfg()` body includes `account_times` object (only non-inherit accounts); switching to inherit omits that key
4. History: each run row shows time + mode (`force` / `scheduled`) + counts; **expand** shows per-account status/attempts/error from `Summary.Accounts`
5. zh-Hant / en / ja strings for new chrome

- [ ] **Step 1: Failing UI tests** — assert rendered HTML:
  - does **not** contain `data-col="five_hour"` / `col_5h` account quota cells
  - contains markers for account schedule inherit/custom (e.g. `data-testid="account-schedule"`)
  - history expand marker / per-account detail container when `RunHistory` has accounts
- [ ] **Step 2: RED**
- [ ] **Step 3: Implement** redesign + JS `saveCfg` / renderHistory
- [ ] **Step 4: GREEN** — `docker compose exec -T dev go test ./internal/management/ -count=1`

---

### Task 6: Docs (three READMEs)

**Files:** `README.md`, `README.zh-Hant.md`, `README.ja.md`

- [ ] Document `account_times`, inherit rules, global-off behavior, UI notes (no 5h/weekly; history expand)
- [ ] Example YAML/JSON matching the spec

---

### Task 7: Full suite + review + chat summary

- [ ] `docker compose exec -T dev go test ./... -count=1`
- [ ] Comment on #11 with implementation summary (optional until commit approval)
- [ ] Code review (Critical/Important/Minor) in Traditional Chinese
- [ ] Chat summary → **ask** whether to commit on feature branch, open PR, merge, delete branch, release

## Non-negotiables

- No Cloud Agent
- No TDD/implementation before **user confirms this plan**
- No `git commit` until user confirms after summary
- After merge to `main`, delete the feature branch
- Chat via SendToUser in **正體中文**

## Spec coverage checklist

| Spec item | Task |
| --- | --- |
| `account_times` + inherit/empty/prune | 1 |
| Union schedule + slot filter | 2–3 |
| Global off stops all scheduled | existing `Start` + Task 2 uses Enabled |
| Manual full allowlist + history | 3 (unchanged force path) + 5 |
| UI redesign, no 5h/weekly | 5 |
| History by account | 5 |
| README ×3 | 6 |
| Branch/PR/delete | 7 (after user OK) |
