# Running UX + sessionStorage Management Key — design

**Date:** 2026-09-22  
**Issue:** [#15](https://github.com/danielhuang-030/cpa-plugin-codex-selective-ping/issues/15)  
**Status:** Design locked in chat (Y); awaiting plan confirmation before TDD

## Goals

1. Make **running** unmistakable while a run is in progress.
2. Keep Management Key available across **same-tab reload** so status polling does not falsely show「需要 Management Key」.
3. Do **not** persist the key to `localStorage` or disk.

## Locked decisions

| Item | Choice |
| --- | --- |
| Key storage | `sessionStorage` only, key name e.g. `codex-selective-ping:management-key` |
| When to write | On successful read of non-empty key before Save / Run / Poll (any time `key()` returns non-empty) |
| When to restore | Page load: if `#management-key` is empty, fill from sessionStorage |
| When to clear | Never write to localStorage; tab close clears sessionStorage naturally. Optional: do not clear on run end. |
| Running UI | Keep `#running-badge` chip **and** add a prominent rail/banner block (e.g. `#running-banner`) with `running_label` + short `run_polling` hint; visible when running |
| i18n | zh-Hant / en / ja (reuse `running_label`, `run_polling`; add banner hint key if needed) |
| History / runner | Unchanged |

## Behavior

### Session key

```
key() {
  read input; if empty, try sessionStorage restore into input; return trim
}
persistKeyIfPresent() {
  if key non-empty → sessionStorage.setItem(...)
}
```

- Call persist after user triggers Save/Run (and when poll successfully uses a key).
- On DOM ready: restore into the password field if empty.
- Tests must still forbid `localStorage.setItem` for the management key; allow `sessionStorage.setItem` with the dedicated key name.

### Running visibility

- Server render: if `Running`, show badge + banner (no `hidden`).
- `enterRunningMode`: show badge + banner, disable run buttons, start poll.
- When poll sees `running === false`: hide banner/badge (reload already does this).

## Non-goals

- Remember key after browser restart
- Encrypt key at rest
- Change End()/history timing
