# Refresh models via Management Key → api-keys — design

**Date:** 2026-09-23  
**Status:** Implemented (working tree; awaiting commit confirmation)  
**Repo:** `cpa-plugin-codex-selective-ping`  
**Related:** `2026-09-23-models-proxy-default-luna-design.md` (plugin `GET …/models` proxy already shipped; upstream still fails because Management Key / Codex token are not accepted by `/v1/models` the way the CPA homepage succeeds)

## Problem

1. Management Key 預設為空；進頁時沒有 key 就不該（也無法）自動拉模型。
2. 現有 plugin `GET /v0/management/plugins/codex-selective-ping/models` 用 Management Key 或 Codex Access Token 打 `{origin}/v1/models`，常落到 fallback（`warning`）。
3. CPA 管理中心首頁「可用模型清單」是：用 Management Key 讀 `api-keys`，再用**代理 API Key** 打 `/v1/models`。官方 plugin ABI **沒有** `host.models.list`。

## Locked decisions

| Item | Choice |
| --- | --- |
| Architecture | Keep plugin models proxy; extend upstream credential cascade |
| UI trigger | New **「更新模型」** button to the right of `#model-select` |
| Empty Management Key (button) | Same as **儲存設定**：show need-key message in `#result`; do not call API; button stays enabled |
| Empty Management Key (boot) | Silent skip (no spam); keep SSR default option only |
| Session has key | Auto `loadModels()` on boot (keep current intent); button can refresh again |
| Credential order | (1) Management Key → `GET {origin}/v0/management/api-keys` → first non-empty proxy API key → `GET {origin}/v1/models` with that API key; (2) Codex Access Token → `/v1/models`; (3) HTTP **200** + fallback ids + `warning` |
| api-keys URL | `{origin}/v0/management/api-keys` with `Authorization: Bearer <Management Key>` (also accept if host already normalized management auth) |
| api-keys parse | Prefer `{"api-keys":[...]}` string list (CPA `GetAPIKeys`); also accept object entries with `api-key` / `apiKey` / `key` |
| Filter | Existing `internal/modelfilter` on plugin side (unchanged) |
| Response shape | Unchanged OpenAI-like `{ "data": [ { "id": "…" }, … ], "warning"?: "…" }` |
| Total failure | HTTP **200** + fallback: configured `model` if set, plus `gpt-6-luna`; include `warning`; **never** clear plugin config |
| Persist model | Unchanged: user must **儲存設定** to write `model` into config |
| Secrets | Never return Management Key, proxy API keys, or Codex tokens to the browser in the models response |
| i18n | Add `refresh_models` (zh-Hant / en / ja) |
| Out of scope | Browser calling `/v1/models` directly; new API-key input field; changing ping transport; changing default model constant; git commit until sp-flow end + explicit OK |

## Behavior

### Management API (plugin)

- Existing route `GET /v0/management/plugins/codex-selective-ping/models` stays registered and Management-Key gated by CPA.
- Handler continues to read Bearer Management Key from the incoming request and resolve `{origin}` as today.
- Update `fetchFilteredModels` cascade:

  1. If Management Key non-empty: `HTTPDo` `GET {origin}/v0/management/api-keys` with that Bearer (and/or `X-Management-Key` if required by local CPA).
  2. Parse first usable proxy API key; if found, `HTTPDo` `GET {origin}/v1/models` with `Authorization: Bearer <api-key>`.
  3. On failure / empty after filter: try first Codex Access Token → `/v1/models` (existing path).
  4. On total failure: return filtered fallback ids + `warning` summarizing last useful error (e.g. api-keys HTTP status / empty keys / upstream HTTP status), still HTTP 200 from plugin handler.

- Do not treat empty `api-keys` as a hard handler error; continue cascade.
- Do not expose raw key material in JSON.

### Management UI

- Layout: model metric row becomes `#model-select` + button 「更新模型」 immediately to its right (`btn-secondary`, compact).
- Button `onclick` → shared refresh path:
  - If `key()` empty → `#result.textContent = msgNeedKey` (same string as save); return.
  - Else call `loadModels()`; while in flight disable the refresh button (optional short busy label); re-enable in `finally`.
- Boot: if `key()` non-empty after session restore → `loadModels()`; if empty → do nothing to the select beyond SSR option.
- `loadModels()` success: rebuild options from `data[].id`; keep orphan current value; set/clear `#model-select-hint` from `warning`.
- Selecting a model does not auto-save; **儲存設定** remains the persistence path.

### i18n

- Key `refresh_models`:
  - zh-Hant: `更新模型`
  - en: `Refresh models`
  - ja: `モデル更新`

## Non-goals

- Asking the user for a separate OpenAI / proxy API key in this UI
- Replacing Management Key with API Key in the password field
- Changing ping request auth (still Codex account token)
- Expanding curated fallback beyond configured model + `gpt-6-luna`
- Committing / releasing as part of design approval

## Test plan (design-level)

- Parse `api-keys` JSON: string array; object array; empty / missing → no key.
- Cascade: api-key success skips Codex; api-keys fail then Codex success; both fail → fallback + `warning` includes configured model and `gpt-6-luna` (deduped).
- Handler still returns 200 on upstream failure path; does not wipe config.
- UI: HTML contains refresh control next to `#model-select`; i18n key present in zh-Hant/en/ja; no-key button path surfaces need-key copy; `MODELS_URL` unchanged.
- Regression: existing models route registration, modelfilter, default `gpt-6-luna` tests stay green.

## Implementation notes (non-binding)

Likely touch: `internal/management/models.go` (+ tests), `internal/management/ui.go` / `ui_test.go`, `internal/management/i18n.go` (+ tests), README×3 mention of refresh if user-facing docs describe the select. Reuse `hostapi.HTTPDo`, origin helper, `modelfilter`, Codex token helper.

Issue / branch / PR / bump / release: after plan confirm → TDD → implement → review → chat summary → **user confirms commit**, following existing ship path.
