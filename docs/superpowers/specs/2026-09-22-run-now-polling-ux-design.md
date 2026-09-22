# Run-now polling UX — design

**Date:** 2026-09-22  
**Status:** Approved in chat (awaiting plan confirmation before TDD)  
**Related:** delayed history after「立刻執行」; `409` / `a run is already in progress`

## Goal

After「立刻執行」, the management UI must show that a run is in progress, prevent confusing double-clicks, and reload only when the run has finished so the new history row is visible — without changing when history is written on the server.

## Problem (current)

1. `POST …/plugins/codex-selective-ping/run` returns **202** immediately while the runner continues in the background (limited outer retry may wait **60s** × `retry_count`, run timeout up to **20m**).
2. History is appended only in `State.End()` when the whole run completes — not at accept time.
3. `runNow()` reloads the page after a fixed **~1.5s**, often while `running` is still true → history looks unchanged; a second click gets **409**.

## Locked decisions

| Item | Choice |
| --- | --- |
| Strategy | **A — frontend poll** existing status JSON `running` |
| Backend contracts | **Unchanged** (no SSE, no in-progress history row, no runner changes) |
| Poll endpoint | `GET /v0/management/plugins/codex-selective-ping/status` (same auth as today: Bearer management key) |
| Poll interval | **~2.5s** (fixed; not configurable) |
| Reload | Only when poll sees `running === false` (or explicit Refresh) |
| Page load | If server-rendered `running` is true → enter running UI and start polling automatically |
| Double-click / 409 | Enter the same running UI + poll; do **not** POST again while already polling |
| History write timing | Still only on `End()` — out of scope to change |

## Out of scope

- SSE / WebSocket push
- Writing a placeholder「進行中」history entry
- Changing `retry_count`, limited-retry interval, or `RunTimeout`
- Full-page timed reload loop (option B) as the primary mechanism

## UI / UX behavior

### Entering “running” mode

Triggers:

1. Successful `POST …/run` (`accepted: true` / HTTP 202), or
2. `POST …/run` returns **409** / `accepted: false` with already running, or
3. Initial HTML indicates `running` (see bootstrap below).

While in running mode:

- Disable「立刻執行」(and keep it disabled until reload after finish).
- Show a visible「執行中」indicator (reuse / extend `running_label` i18n: zh-Hant / en / ja).
- Show a short status line in the existing result area (e.g. polling / waiting for finish) — also i18n.
- Start (or continue) polling; do not fire another `POST /run`.

### Polling

1. `GET …/status` with `Authorization: Bearer <management-key>` every ~2.5s.
2. If response JSON has `running: true` → keep polling.
3. If `running: false` → `location.reload()` once (history should now include the completed run).
4. If management key is empty → do not poll; show existing need-key message (same as save/run today).
5. Transient fetch errors → keep polling (bounded patience is fine; no hard fail that clears the lock UI); optional brief error text in the result area.

### Bootstrap on page load

- Server-rendered HTML exposes current `StatusResponse.Running` (e.g. `data-running="true|false"` on a root element, or a small inline `const initialRunning=…`).
- On `DOMContentLoaded` (or end of script): if `initialRunning` and key present → enter running mode and poll. If key missing → show running indicator + need-key for poll (still disable Run now so the user does not stack POSTs blindly).

### Remove

- Fixed `setTimeout(…reload, 1500)` after accepted run.

## i18n

Required keys (zh-Hant / en / ja), names illustrative:

- `running_label` — already present; wire into UI if unused
- `run_busy` / `run_polling` — short copy for result area while waiting
- Optional: `run_already` — when 409 is shown before polling continues

## Testing

Prefer existing management UI string/HTML tests (no browser driver required):

1. Rendered page with `Running: true` includes bootstrap marker (`data-running="true"` or equivalent) and disables run button (or marks it disabled in markup).
2. Embedded script no longer contains the fixed 1500ms reload-after-run pattern; contains poll-against-`/status` and reload-on-`running` false behavior (assert distinctive substrings).
3. i18n maps contain the new keys for all three languages.
4. Handler/status JSON still exposes `running` (existing tests suffice; no contract change).

## Non-goals for verification

- Live CPA against real Codex accounts (optional smoke later).
- Changing history persistence format.

## Success criteria

- User clicks「立刻執行」→ sees執行中, button locked.
- After the background run finishes (including limited retries), page reloads and the new history entry is visible without a manual wait-and-guess refresh.
- Refresh mid-run still shows執行中 and resumes polling when a management key is available.
- Second click during a run does not confuse; UI stays in running mode.
