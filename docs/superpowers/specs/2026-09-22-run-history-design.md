# Run history persistence — design

**Issue:** [#9](https://github.com/danielhuang-030/cpa-plugin-codex-selective-ping/issues/9)  
**Date:** 2026-09-22  
**Status:** Approved

## Goal

Replace single-summary `last_run.json` overwrite with a bounded execution history list. Management UI keeps the latest “receipt” and adds a history list below.

## Locked decisions

| Item | Choice |
| --- | --- |
| Config key | `history_limit` (default **60**) |
| UI | Latest receipt + history list |
| File | New `run_history.json` |
| Migration | If only legacy `last_run.json` → migrate into history → **delete** `last_run.json` |
| Invalid / omitted limit | Treat as 60; clamp configured values to ≥ 1 |

## On-disk format

Path resolution (unchanged rules, new filename):

- Default: `{CPA root}/data/codex-selective-ping/run_history.json`
- `data_dir` set → `{data_dir}/run_history.json`
- `state_path` set → that exact file path (history file)

```json
{
  "version": 1,
  "runs": [ /* newest first; each entry = existing Summary shape */ ]
}
```

Write path: prepend new run, trim to `history_limit`, atomic write (tmp + rename) as today.

## Runtime / API

- In-memory: keep full trimmed `runs` slice; `last_run` / `LastRun` remains `runs[0]` for compatibility.
- Status JSON: keep `last_run`; add `run_history` array (same order as disk).
- Persist after every completed run (`State.End`).

## Migration

On `SetPersistPath` / load:

1. If `run_history.json` (or configured `state_path`) exists → load `version` + `runs`.
2. Else if sibling/default `last_run.json` exists with a single `Summary` → wrap as one-element `runs`, write history file, delete `last_run.json`.
3. Corrupt / missing → empty history (no panic).

## UI

- Keep current receipt for latest run.
- Below: compact list (timestamp, mode, success/fail counts); optional expand for per-account detail.
- Empty state when no runs.
- i18n: zh-Hant / en / ja.

## Tests

- Migrate legacy file; old deleted; new file valid.
- Append + trim at limit.
- Reload after restart.
- Config default 60; custom limit; clamp.
- Status exposes `last_run` + `run_history`.
- UI contains history list markers / i18n keys.

## Out of scope

Export/CSV, search/filter, per-account aggregation, Cloud Agent, version bump/release (unless asked later).

## Approach

Extend `internal/runstate` persistence + config/registration + management handler/UI/i18n + README config table. No new top-level package.
