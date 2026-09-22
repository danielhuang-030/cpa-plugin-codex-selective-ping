# Running button + mobile RWD — design

**Date:** 2026-09-22  
**Issue:** [#17](https://github.com/danielhuang-030/cpa-plugin-codex-selective-ping/issues/17)  
**Status:** Awaiting plan confirmation

## Problems

1. While a run is active,「立刻執行」still looks clickable; users want it to become「執行中」and gray/disabled.
2. Buttons lack a pressed feel (`:hover` / `:active`).
3. On mobile, schedule time cards float over the middle of the UI (over the rail between status and Management Key). Root cause: `.rail { position: sticky }` remains on when `.shell` is a single column, so the workspace `.timeline` scrolls under/over the sticky rail.

## Locked decisions

| Item | Choice |
| --- | --- |
| Idle button label | Keep `run_now`（立刻執行 / Run now / 今すぐ実行） |
| Running button label | Switch to `running_label`（執行中 / Running / 実行中） on all `[data-run-now]` buttons |
| Running control | `disabled` + gray CSS (opacity / muted border / no pointer) |
| Press feedback | Add `:hover` and `:active` (translate/brightness) for `.btn-primary`, `.btn-secondary`, `.btn-ghost`; disabled skips press styles |
| Chip `#running-badge` | **Keep** (small status) |
| Banner `#running-banner` | **Keep** (extra clarity); button label change is the primary control affordance |
| Mobile rail | `@media (max-width: 960px) { .rail { position: static; } }` |
| Key / history / retry | Unchanged |

## Behavior

- `setRunningUI(true)` / `enterRunningMode`: disable run buttons, set `textContent` / visible label to `running_label` (store idle label in `data-idle-label` on first run), show badge+banner, start poll.
- `setRunningUI(false)` (after reload normally): restore idle label from `data-idle-label`.
- Server render when `Running`: buttons already `disabled` with running label text.

## Non-goals

- Redesigning the whole mobile IA
- Changing CPA chrome (hamburger / theme icons come from host shell)
