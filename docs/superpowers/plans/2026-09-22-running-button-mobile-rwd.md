# Running button + mobile RWD — implementation plan

> **Do not start TDD until the user confirms this plan in chat.**

**Goal:** Running state on the Run button + press feedback + fix mobile sticky-rail overlap (#17).

**Architecture:** Management UI only (`ui.go` / `ui_test.go`; i18n reuse). No runner changes.

**Spec:** `docs/superpowers/specs/2026-09-22-running-button-mobile-rwd-design.md`

## Global constraints

- No Cloud Agent; 正體中文 chat; no commit until end-of-flow confirmation
- Docker: `sudo docker compose exec -T dev go test …`

---

### Task 1: RED tests

**Files:** `internal/management/ui_test.go`

- [ ] Assert CSS contains `@media (max-width: 960px)` rule with `.rail` `position: static` (or `sticky` disabled)
- [ ] Assert JS/HTML: running mode sets run button label via `running_label` / `data-idle-label` markers
- [ ] Assert `:active` (or press) styles for `.btn-primary` / `.btn-secondary`
- [ ] When `Running: true`, `[data-run-now]` is `disabled` and button text includes running label (zh-Hant「執行中」)
- [ ] RED in Docker

### Task 2: Mobile RWD GREEN

**Files:** `internal/management/ui.go` CSS

- [ ] Under existing `@media (max-width: 960px)` (or sibling): `.rail { position: static; top: auto; }`

### Task 3: Running button label + disabled GREEN

**Files:** `internal/management/ui.go` JS/HTML

- [ ] On render: each `[data-run-now]` gets `data-idle-label` with current `run_now` text
- [ ] If `Running`, set button text to `running_label` and disabled
- [ ] `showRunningBadge` / `enterRunningMode`: update all `[data-run-now]` text to running label; on disable restore idle label when turning off (reload usually resets)

### Task 4: Press feedback GREEN

**Files:** `internal/management/ui.go` CSS

- [ ] `:hover` / `:active` for primary/secondary/ghost; `:disabled` muted + `cursor: not-allowed` + no active transform

### Task 5: Full suite + review gate

- [ ] `sudo docker compose exec -T dev go test ./... -count=1`
- [ ] 正體中文 summary + ask before commit / PR / release

## Non-negotiables

- No TDD before plan confirmation
- No commit until user confirms after summary
