# Limited-quota outer retry — implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:test-driven-development. Steps use checkbox (`- [ ]`) syntax. **Do not start TDD until the user explicitly confirms this plan in chat.**

**Goal:** Add `retry_count` (default 2): on `limited` only, wait fixed 60s and retry up to that many times per account per trigger; stop at cap. Issue #10.

**Architecture:** Config + registration + runner outer loop with injectable `sleep` for tests. Inner pinger HTTP retries unchanged. Docs in three READMEs.

**Tech stack:** Go, existing Docker Compose `dev` service. **No git commit** until end-of-flow user confirmation.

**Spec:** `docs/superpowers/specs/2026-09-22-limited-retry-design.md`

---

### Task 1: Config — `retry_count`

**Files:**
- Modify: `internal/config/config.go`, `internal/config/config_test.go`
- Modify: `registration.go`

- [ ] **Step 1: Failing tests** — omit → 2; `retry_count: 5` → 5; negative → 0 (or clamp per Validate)
- [ ] **Step 2: RED in Docker** — `docker compose exec -T dev go test ./internal/config/ -count=1`
- [ ] **Step 3: Implement** field + JSON/YAML parse + DefaultConfig + registration ConfigField
- [ ] **Step 4: GREEN**

---

### Task 2: Runner outer limited-retry helper (TDD)

**Files:**
- Modify: `internal/runner/runner.go`
- Add/Modify: `internal/runner/runner_test.go` (or `retry_test.go`)

- [ ] **Step 1: Failing tests** with fake pinger / host and **injectable sleep** (record sleep durations; advance immediately):
  - limited → sleep 60s → success; attempts/calls == 2 when `retry_count=2`? Wait: first limited, one retry success → 2 PingAccount calls, one 60s sleep, `retry_count` remaining after first fail = 2 so could retry twice — after first retry success stop. So calls=2, sleeps=1.
  - limited ×3 with `retry_count=2` → 3 calls, 2 sleeps, final limited
  - `retry_count=0` + limited → 1 call, 0 sleeps
  - failed (non-limited) → 1 call, 0 sleeps
- [ ] **Step 2: RED**
- [ ] **Step 3: Implement** `pingWithLimitedRetry(ctx, …, retryCount, sleep fn)` used from account goroutine; honor `ctx.Done()` during wait
- [ ] **Step 4: Wire** `cfg.RetryCount` from plugin/runner entry
- [ ] **Step 5: GREEN** — do not real-sleep 60s in tests

---

### Task 3: Docs

**Files:** `README.md`, `README.zh-Hant.md`, `README.ja.md`

- [ ] Document `retry_count` in config example + reference table (default 2, 60s, limited-only)

---

### Task 4: Full suite + review + chat summary

- [ ] `docker compose exec -T dev go test ./... -count=1`
- [ ] Comment on #10 with progress
- [ ] Code review + Traditional Chinese summary → **ask before commit**

## Non-negotiables

- No Cloud Agent
- No TDD/implementation before **user confirms this plan**
- No `git commit` until user confirms after summary
- Chat via SendToUser in **正體中文**
