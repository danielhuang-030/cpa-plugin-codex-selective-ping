# Limited-quota outer retry — design

**Issue:** [#10](https://github.com/danielhuang-030/cpa-plugin-codex-selective-ping/issues/10)  
**Date:** 2026-09-22  
**Status:** Approved (awaiting plan confirmation before TDD)

## Goal

When a ping fails due to **quota / rate limit (`limited`)**, retry with a fixed interval so a short timing gap (e.g. slot at 11:00 empty, quota restored at 11:01) does not skip the entire schedule window.

## Locked decisions

| Item | Choice |
| --- | --- |
| Config key | `retry_count` |
| Default | `2` → up to **2 retries after the first attempt** (max **3** requests per account per trigger) |
| Interval | Fixed **60 seconds** (not configurable) |
| Trigger | Only status **`limited`** (429 / usage-limit style) |
| `resets_at` | **Ignored** for wait timing; still sleep fixed 60s |
| Cap | Stop when retries exhausted; next schedule slot unchanged |
| Scope | Scheduled and manual “run now” |
| `retry_count: 0` | No outer retry (first `limited` ends that account for this trigger) |

## Relationship to existing pinger retries

- **Inner** (`pinger`): short exponential backoff for retryable HTTP (non-`limited`), max 3 — **unchanged**.
- **Outer** (this feature): minute-level loop only when outcome is `limited`, controlled by `retry_count`.

## Runtime behavior (per account)

1. Call `PingAccount` (inner logic as today).
2. If not `limited` → done (success / failed / deferred / …).
3. If `limited` and outer retries remaining → wait 60s (respect run `context` cancel / 20m timeout) → retry.
4. If `limited` and no retries left → record `limited` and stop.

## Config / registration / docs

- Add `RetryCount int` on config with default 60→ wait, default **2**; clamp negatives to 0.
- Expose in `registration` ConfigFields.
- README en / zh-Hant / ja.

## Tests

- `limited` then success after one outer wait.
- Exhaust `retry_count` still `limited`.
- `retry_count: 0` → single attempt.
- Non-`limited` failure does not take outer path.
- Prefer injectable sleep / fake clock in tests (do not sleep real 60s in CI).

## Out of scope

Smart wait until `resets_at`, configurable interval, outer retry for non-limited errors.

## Approach

Outer loop in `internal/runner` around `pinger.PingAccount` (or small helper), not inside pinger inner HTTP retry loop. Pass `cfg.RetryCount` into runner.
