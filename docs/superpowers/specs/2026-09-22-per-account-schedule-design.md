# Per-account schedule + Management UI redesign — design

**Issue:** [#11](https://github.com/danielhuang-030/cpa-plugin-codex-selective-ping/issues/11)  
**Date:** 2026-09-22  
**Status:** Approved (brainstorm §1–§3)

## Goal

1. Each selected Codex account may use its own daily `times`, or inherit the global unified schedule.
2. Fully redesign the Management UI information architecture and presentation.
3. Remove 5h / weekly quota columns from the account UI.
4. Make run history richer and expandable **per account**; manual runs remain in history (already persisted via `State.End`).

## Locked decisions

| Item | Choice |
| --- | --- |
| Per-account override | **`times` only** |
| Timezone | Always **global** |
| Inherit rule | Missing / empty custom → use global `times` |
| Global off | `schedule_enabled=false` stops **all** scheduled fires (including custom); manual still works |
| Config shape | Keep `accounts: []string`; add `account_times` map (Approach A) |
| Scheduler | Single timer; union of effective times; on fire ping only accounts whose effective times contain that `HH:MM` |
| Orphan `account_times` keys | **Prune on save** if not in allowlist |
| Empty custom list | Treat as inherit (same as no key); not “never ping” |
| Quota UI | Do **not** show 5h / weekly on account cards |
| History | Expandable per-run detail by account; modes include `manual` and schedule |
| Git workflow | Feature branch + PR; delete branch after merge to `main` |
| Out of scope | Per-account timezone / per-account schedule toggle; object-shaped `accounts`; CSV/search; changing limited-retry (#10) semantics |

## Config

### Fields

| Key | Type | Role |
| --- | --- | --- |
| `schedule_enabled` | bool | Master schedule switch |
| `timezone` | string | Global IANA timezone |
| `times` | `[]string` (`HH:MM`) | Unified default daily slots |
| `accounts` | `[]string` | Allowlist (unchanged) |
| `account_times` | `map[string][]string` | Optional per-account `times` override |

### Example

```yaml
schedule_enabled: true
timezone: Asia/Taipei
times: ["06:00", "11:00", "16:00", "21:00"]
accounts:
  - alice@example.com
  - bob@example.com
account_times:
  bob@example.com: ["07:30", "19:00"]
```

Effective schedules: alice → global four slots; bob → 07:30 and 19:00 only.

### Validation

- Same `HH:MM` rules as global `times`.
- Invalid custom entry → save rejected.
- On save: drop `account_times` keys not present in `accounts`.

### Compatibility

- Absent `account_times` → behavior identical to today (everyone uses global `times`).
- CPA Management API / config decode paths must round-trip the new field (JSON + YAML subset as applicable).

## Scheduler

1. If `!schedule_enabled` → stop; clear next-run.
2. Build the set of allowlisted accounts.
3. For each, `effectiveTimes(id) = account_times[id] if non-empty else global times`.
4. `nextFire` = earliest future clock among the **union** of those effective times (in global timezone).
5. On fire at local `HH:MM`: run only allowlisted accounts whose `effectiveTimes` contain that `HH:MM`.
6. Reschedule from “now” after each fire (same loop pattern as today).

Manual run: unchanged target set (current allowlist / existing manual semantics), not filtered by “this tick”.

## Runner / state

- Continue building `Summary` with `Mode` (`manual` | schedule) and per-account `AccountResult` rows; call `State.End` so history persists.
- Schedule path must accept a **subset** of allowlist (tick targets), not always full allowlist.
- Prefer reusing existing result fields; add fields only if UI detail is missing.

## Management UI (full redesign)

### Information architecture

1. **Rail / top:** plugin status, next run, allowlist counts, timezone, management key, Save / Run now / Refresh.
2. **Global rhythm:** schedule master switch, timezone, unified `times` timeline (default for “inherit” accounts).
3. **Accounts (primary):** selectable cards/rows; each account: inherit vs custom + custom times editor; **no 5h / weekly**; light status / plan optional.
4. **Run + history:** latest receipt; history list with expand → per-account rows (status, attempts, error, etc.); zh-Hant / en / ja.

### Interactions

- Switching an account from custom → inherit clears that key from `account_times` on save.
- History empty state + i18n for mode labels and expand detail.

### Visual direction

- Full-page redesign (not a small patch on the current grid).
- Keep operational clarity (status, next run, actions) over decorative chrome.
- Prior Taste/ops-console work is reference only; this feature may reshape layout around account-centric schedule editing.

## Testing focus

- Config: parse / validate `account_times`; inherit vs override; prune orphans on save.
- Scheduler: union `NextRun`; fire filters to matching accounts only; global off disables all.
- Runner: scheduled subset; manual still ends in history with `mode=manual`.
- UI/tests: no 5h/weekly markers in account section; history expand shows per-account detail; i18n keys present.

## Docs

- Update README.md / README.zh-Hant.md / README.ja.md: new key, inherit rules, UI notes.

## Implementation approach

Extend existing packages (`config`, `scheduler`, `runner`, `runstate`, `management`) rather than a new top-level package. Feature branch + PR for #11; merge to `main` then delete branch. Version bump / release only if requested after merge.
