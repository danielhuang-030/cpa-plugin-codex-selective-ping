# Models list proxy + default `gpt-6-luna` — design

**Date:** 2026-09-23  
**Status:** Awaiting user review of written spec  
**Related:** Approach A in `2026-09-23-model-select-quota-blurb-design.md` (browser `GET /v1/models`) now fails with **HTTP 401** when using Management Key; this spec activates the documented Approach B fallback and updates the built-in default model.

## Problems

1. Management UI `loadModels()` calls `GET /v1/models` with the Management Key Bearer. CPA returns **401 Unauthorized**, so the model `<select>` cannot refresh options (current value is kept; hint shows `models: HTTP 401`).
2. Built-in fallback / default ping model is still `gpt-5.6-luna` (`pinger.ModelName`). Product wants the default to be **`gpt-6-luna`**.

## Locked decisions

| Item | Choice |
| --- | --- |
| Architecture | **Approach B:** plugin management endpoint proxies model list; browser stops calling `/v1/models` directly |
| New endpoint | `GET /v0/management/plugins/codex-selective-ping/models` (same Management Key auth as status / run / config) |
| Upstream URL | Same origin as the incoming management request: build from `X-Forwarded-Proto` + `X-Forwarded-Host` when present, else request `Host` → `{origin}/v1/models` |
| Upstream auth order | (1) forward Management Key Bearer; (2) on non-2xx / hard failure, retry once with first available Codex account AccessToken from host auth |
| Filter | Existing `internal/modelfilter` heuristics (Codex / OpenAI / ChatGPT / `gpt-` style); apply on the plugin side before responding |
| Response shape | OpenAI-like `{ "data": [ { "id": "…" }, … ] }` plus optional `warning` string when serving fallback or degraded upstream |
| Total upstream failure | HTTP **200** to the browser with fallback ids: configured `model` if set, plus default `gpt-6-luna`; include `warning`; **do not** clear plugin config |
| No second credential | If no Codex auth material exists, only try Management Key, then fallback |
| Secrets | Never return AccessToken (or raw upstream auth headers) to the browser |
| Orphan selection | If configured `model` is missing from filtered upstream list, still include it as an option (unchanged contract) |
| Default model | Change `pinger.ModelName` (and all docs / tests that assert the built-in default) from `gpt-5.6-luna` → **`gpt-6-luna`** |
| Empty config `model` | Effective model remains: trim(`cfg.Model`) if non-empty, else `pinger.ModelName` (`gpt-6-luna`) |
| Save UX | Unchanged: select change → existing `saveCfg()` → `PATCH .../config` |
| History pagination | Out of scope |

## Behavior

### Management API

- Register / handle `GET .../plugins/codex-selective-ping/models` alongside existing status / run handlers.
- Require the same management authentication CPA already enforces for other plugin management routes (browser continues to send Management Key).
- Resolve upstream base URL from the inbound request (forwarded headers preferred).
- `HTTPDo` `GET {origin}/v1/models` with Authorization attempt (1), then optional attempt (2).
- Parse OpenAI-style `data[]` (at least `id`; use `owned_by` when present for filtering).
- Return filtered `{ "data": [ { "id" }, … ] }`. On degraded path set `warning` (e.g. upstream status or short reason). Prefer **200 + fallback** over propagating upstream 401/5xx to the UI, unless the handler itself errors (bad internal state).

### Management UI

- `loadModels()` fetches the new plugin models URL with Management Key (not `/v1/models`).
- Rebuild `<select>` from `data[].id`; keep orphan current value; clear hint on clean success.
- If `warning` is present or request fails at the plugin layer: show non-blocking hint; keep usable select (fallback / prior options); do not clear config.
- If Management Key is empty: keep current behavior (skip fetch / do not wipe options).

### Default model

- `pinger.ModelName = "gpt-6-luna"`.
- README (en / zh-Hant / ja) and any user-facing copy that names the built-in default update accordingly.
- Tests that hard-code `gpt-5.6-luna` as the built-in default update to `gpt-6-luna`.

## Non-goals

- Adding a user-supplied OpenAI API key field in the management UI
- Changing the ping transport (still Codex URL + account AccessToken)
- Auto-selecting a “best” model
- CPA host/global model setting
- History list pagination changes
- Committing / releasing as part of design approval (sp-flow: commit only after full flow + explicit user OK)

## Test plan (design-level)

- Origin builder: Host only; `X-Forwarded-Proto` + `X-Forwarded-Host`.
- Auth sequence: Management Key success (no Codex retry); Management Key 401 then Codex token success; both fail → 200 + fallback ids + `warning`.
- Fallback contents: includes configured model when set; always includes `gpt-6-luna` when using fallback path (deduped).
- Filter: regression on sample payloads via `modelfilter`.
- UI tests: JS references plugin models path (not bare `/v1/models`); hint / orphan behavior assertions updated as needed.
- Default: `EffectiveModel` empty → `gpt-6-luna`; pinger/constant tests updated.

## Implementation notes (non-binding)

Likely touch: `internal/management/handler.go` (+ tests), `internal/management/ui.go` / `ui_test.go`, `internal/pinger` constant (+ tests), README×3, possibly a small helper for origin + upstream fetch; reuse `internal/modelfilter` and host `AuthList` / `AuthGet` / `HTTPDo`.

Issue / branch / PR / bump / release: after plan confirm → TDD → implement → review → chat summary → **user confirms commit**, following existing ship path.
