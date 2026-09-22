# Remove quota blurb + selectable ping model — design

**Date:** 2026-09-23  
**Issue:** [#21](https://github.com/danielhuang-030/cpa-plugin-codex-selective-ping/issues/21)  
**Status:** Awaiting user review of written spec

## Problems

1. The「怎麼算到點？」help list still shows a「額度欄 / 已從帳號卡移除」row. That narrative is obsolete and should go.
2. Ping model is a hard-coded constant (`pinger.ModelName` = `gpt-5.6-luna`). The「此刻」rail only displays it. Users want to **choose** the model; choices come from CPA.

## Locked decisions

| Item | Choice |
| --- | --- |
| Quota help row | Remove the `how_quota_removed` / `how_quota_removed_v` metric row (and stop requiring those i18n keys in UI tests) |
| Selected model persistence | Plugin config field `model` (string), saved via existing Management UI `saveCfg()` → `PATCH .../config` |
| Picker placement | Only in「此刻」rail: replace static model text with a `<select>` |
| Model list source | Browser `GET /v1/models` (OpenAI-compatible CPA endpoint) |
| List filter | Keep only Codex / OpenAI / ChatGPT–related entries (id / owned_by heuristics; case-insensitive) |
| Architecture | **Approach A:** browser fetches `/v1/models`, select in rail, persist with existing config save — no plugin proxy unless `/v1/models` proves unreachable from the management page |
| Empty `model` | Ping and display fall back to `pinger.ModelName` (`gpt-5.6-luna`) |
| Orphan selection | If configured `model` is missing from the filtered list, still show it as an option so the value is not silently dropped |
| Other locales | zh-Hant / en / ja stay in sync for any copy changes |

## Behavior

### Quota blurb

- Server-rendered HTML for the「怎麼算到點？」block no longer includes the quota-removed metric row.
- Tests that currently assert presence of `how_quota_removed` keys update to assert absence from that block (or drop those keys from the required-key list if unused elsewhere).

### Model select

-「此刻」model row renders a `<select>` instead of plain text.
- On load (and optionally on focus/open), JS calls `GET /v1/models` with the best auth the management page can use for OpenAI-compatible routes (prefer the same Bearer already used for management calls if CPA accepts it). If the call fails: keep current value as the only option, show a non-blocking error hint, do not clear config.
- Filter candidate models: include when `id` or `owned_by` (or equivalent fields) matches heuristics such as containing `codex`, `openai`, `chatgpt`, or `gpt-` (case-insensitive). Tune only if real CPA payloads need it.
- Changing the select updates the in-memory config payload’s `model` and saves through existing `saveCfg()` (same UX as other settings — no dedicated “save model only” API).
- Status / rail display of “current model” uses effective model: `cfg.Model` if non-empty, else `pinger.ModelName`.
- Pinger request body `model` uses that same effective value.

### Config

- Add optional `model` string to `internal/config.Config` (JSON/YAML parse + validate; empty allowed).
- Default config leaves `model` empty (meaning “use built-in default”). Empty → `pinger.ModelName` is the contract.

## Non-goals

- Plugin backend proxy for `/v1/models` (fallback only if Approach A cannot work)
- CPA global / host-wide model setting
- Restoring quota columns on account cards
- Changing ping prompt text beyond the `model` field
- Auto-picking “best” model

## Test plan (design-level)

- Config: parse/round-trip `model`; empty → effective default.
- Filter helper: include/exclude sample `/v1/models` payloads.
- Pinger: uses configured model; empty falls back to `ModelName`.
- UI/i18n: help block has no quota-removed row; rail exposes a select (markup assertion).
- Manual: load options, change, save, reload persists; failed `/v1/models` still shows prior value.

## Implementation notes (non-binding)

Likely touch: `internal/config`, `internal/pinger`, `internal/management` (`ui.go`, `i18n.go`, handler status `Model` field), README config table, unit tests beside those packages.
