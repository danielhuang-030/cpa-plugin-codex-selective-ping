# Models list proxy + default `gpt-6-luna` — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.  
> **Do not start TDD until the user confirms this plan in chat.**  
> **sp-flow:** No `git commit` / push / PR / release until end-of-flow user confirmation. Chat in 正體中文. No Cloud Agent. Tests via Docker Compose `dev`.

**Goal:** Stop the Management UI from calling CPA `GET /v1/models` directly (401 with Management Key); serve a filtered model list from a new plugin management endpoint that tries Management Key then the first Codex AccessToken; change the built-in default model to `gpt-6-luna`.

**Architecture:** Approach B — browser calls `GET /v0/management/plugins/codex-selective-ping/models` with Management Key. Handler builds same-origin `{origin}/v1/models` from `X-Forwarded-*` / `Host`, `HTTPDo`s with auth cascade, filters via `modelfilter.Keep`, returns OpenAI-like `{data:[{id}], warning?}`. Total upstream failure → HTTP 200 + fallback ids (configured `model` + `gpt-6-luna`) + `warning`. Never return tokens to the browser.

**Tech Stack:** Go plugin (`internal/management`, `internal/pinger`, `internal/modelfilter`, `internal/hostapi`, `internal/selector`), Management UI inline JS, Docker Compose service `dev` for `go test`.

**Spec:** `docs/superpowers/specs/2026-09-23-models-proxy-default-luna-design.md`

## Global Constraints

- Chat / progress: 正體中文
- Tests: `docker compose exec -T dev go test ./… -count=1` (narrow `-run` while iterating)
- No Cloud Agent
- **No git commit / push / PR / release until user confirms after review**
- Do not add a user-supplied API key field
- Do not change ping transport (still Codex URL + AccessToken)
- Do not change history pagination
- Upstream auth order locked: Management Key → first Codex AccessToken → fallback
- Base URL locked: same origin from inbound request
- Default constant locked: `pinger.ModelName = "gpt-6-luna"`

## File map

| File | Role |
| --- | --- |
| `internal/pinger/pinger.go` | Change `ModelName` to `gpt-6-luna` |
| `internal/pinger/*_test.go` / `internal/config/*_test.go` / `internal/management/*_test.go` / `internal/modelfilter/filter_test.go` | Update literals that assert the **built-in default** (not unrelated fixture model ids unless they copy the constant) |
| `internal/management/origin.go` (new) | Build `{scheme}://{host}` from request headers |
| `internal/management/origin_test.go` (new) | Origin builder unit tests |
| `internal/management/models.go` (new) | Fetch/filter/fallback helpers + response types |
| `internal/management/models_test.go` (new) | Auth cascade + fallback tests (mock host) |
| `internal/management/handler.go` | Route `GET …/models` |
| `internal/management/handler_test.go` | Extend mock host `HTTPDo`; route integration tests |
| `internal/management/ui.go` | `loadModels()` → plugin models URL; honor optional `warning` |
| `internal/management/ui_test.go` | Assert JS uses plugin path, not bare `/v1/models` |
| `README.md` / `README.zh-Hant.md` / `README.ja.md` | Default `gpt-6-luna`; note list via plugin proxy |

---

### Task 1: Default model → `gpt-6-luna`

**Files:**
- Modify: `internal/pinger/pinger.go` (`ModelName` const)
- Modify: tests/READMEs that hard-code the built-in default string `gpt-5.6-luna` (search repo; update assertions / docs that mean “builtin default”)
- Prefer tests that already compare to `pinger.ModelName` — those pass automatically after const change

**Produces:** Empty config `model` → effective `gpt-6-luna`.

- [ ] **Step 1: Failing test** — if any test hard-asserts the new default before the const changes, add/adjust one explicit check:

```go
func TestModelNameIsGPT6Luna(t *testing.T) {
	if ModelName != "gpt-6-luna" {
		t.Fatalf("ModelName=%q want gpt-6-luna", ModelName)
	}
}
```

(Place in `internal/pinger/pinger_test.go`.)

- [ ] **Step 2: RED** — `docker compose exec -T dev go test ./internal/pinger/ -run TestModelNameIsGPT6Luna -count=1`  
  Expected: FAIL (`got gpt-5.6-luna` or current value).

- [ ] **Step 3: Implement** — set `ModelName = "gpt-6-luna"` in `pinger.go`.

- [ ] **Step 4: GREEN** — same test; then update remaining hard-coded **default** literals in tests/READMEs (`rg 'gpt-5\.6-luna'`) so `./...` stays green. Fixture values that intentionally pin a *configured* model may stay if not documenting the builtin default — but if they copied the old default for “empty → fallback” docs/examples, switch them.

- [ ] **Step 5:** Do **not** commit.

---

### Task 2: Same-origin URL builder

**Files:**
- Create: `internal/management/origin.go`
- Create: `internal/management/origin_test.go`

**Produces:**

```go
// originFromHeaders returns scheme://host with no trailing slash.
// Prefer X-Forwarded-Proto + X-Forwarded-Host when both (or host) present;
// else scheme defaults to "http" when only Host is available without forwarded proto.
func originFromHeaders(headers map[string][]string) string
```

Header lookup must be case-insensitive (reuse existing `firstHeader` helpers in the package if present).

- [ ] **Step 1: Failing tests**

```go
func TestOriginFromHeadersHostOnly(t *testing.T) {
	got := originFromHeaders(map[string][]string{"Host": {"cpa.example:8317"}})
	if got != "http://cpa.example:8317" {
		t.Fatalf("got %q", got)
	}
}

func TestOriginFromHeadersForwarded(t *testing.T) {
	got := originFromHeaders(map[string][]string{
		"X-Forwarded-Proto": {"https"},
		"X-Forwarded-Host":  {"cpa.example"},
		"Host":              {"localhost:8317"},
	})
	if got != "https://cpa.example" {
		t.Fatalf("got %q", got)
	}
}
```

- [ ] **Step 2: RED** — `docker compose exec -T dev go test ./internal/management/ -run TestOriginFromHeaders -count=1`

- [ ] **Step 3: Implement** `originFromHeaders` as specified.

- [ ] **Step 4: GREEN** — same command.

- [ ] **Step 5:** Do **not** commit.

---

### Task 3: Models fetch helpers (auth cascade + fallback)

**Files:**
- Create: `internal/management/models.go`
- Create: `internal/management/models_test.go`
- Modify: `internal/management/handler_test.go` mock host **or** define a dedicated mock in `models_test.go` that records `HTTPDo` Authorization headers and returns scripted responses

**Produces (exact contracts):**

```go
type modelsListResponse struct {
	Data    []modelsListItem `json:"data"`
	Warning string           `json:"warning,omitempty"`
}
type modelsListItem struct {
	ID string `json:"id"`
}

// bearerFromAuthHeader parses "Bearer <token>" (case-insensitive Bearer).
func bearerFromAuthHeader(headers map[string][]string) string

// firstCodexAccessToken lists auth files, keeps selector.IsCodex, AuthGet + parse token.
// Reuse pinger token parsing: either export a small helper from pinger
// (e.g. AccessTokenFromAuthJSON([]byte) (string, error)) or duplicate the
// minimal top-level access_token read in models.go — prefer exporting from pinger
// if it avoids drift.
func firstCodexAccessToken(ctx context.Context, h hostapi.Host) (string, error)

// fetchFilteredModels:
// 1) GET origin+"/v1/models" with Bearer managementKey if non-empty
// 2) on non-2xx or transport error, retry once with Codex token if available
// 3) parse OpenAI data[].id (+ owned_by when present), keep modelfilter.Keep
// 4) if still no usable list: return fallback IDs (cfgModel if set, plus pinger.ModelName), deduped, with warning
// Never include tokens in the returned struct.
func fetchFilteredModels(ctx context.Context, h hostapi.Host, origin, managementKey, cfgModel string) modelsListResponse
```

- [ ] **Step 1: Failing tests** (table-driven or separate funcs):

  1. Management Key 200 → uses that Authorization only; returns filtered ids; `Warning == ""`; Codex path not required.
  2. Management Key 401 then Codex token 200 → second `HTTPDo` uses Codex Bearer; returns filtered ids.
  3. Both fail → `Data` contains configured model (e.g. `gpt-5.6-luna`) and `gpt-6-luna`; `Warning != ""`; Status path tested at handler layer in Task 4.
  4. Filter drops unrelated ids (e.g. `claude-3` without matching owned_by).

- [ ] **Step 2: RED** — `docker compose exec -T dev go test ./internal/management/ -run 'TestFetchFiltered|TestBearer|TestFirstCodex' -count=1`

- [ ] **Step 3: Implement** helpers. For `HTTPDo` request shape match `hostapi.HTTPRequest` / existing pinger usage (`Method`, `URL`, `Headers` with `Authorization: []string{"Bearer " + token}`).

- [ ] **Step 4: GREEN** — same test command.

- [ ] **Step 5:** Do **not** commit.

---

### Task 4: Wire `GET …/models` on Handler

**Files:**
- Modify: `internal/management/handler.go`
- Modify: `internal/management/handler_test.go`

**Produces:** New switch case:

- Method `GET` and path suffix `/plugins/codex-selective-ping/models` (same matching style as status/run).
- Call `fetchFilteredModels` with `originFromHeaders(req.Headers)`, `bearerFromAuthHeader(req.Headers)`, `config.Model` from `h.Plugin.Config()` (raw config model string, not necessarily EffectiveModel — fallback helper adds `pinger.ModelName`).
- Respond `200` + `application/json` body of `modelsListResponse` (including degraded fallback).
- Do not return tokens.

- [ ] **Step 1: Failing tests**

```go
func TestModelsEndpointManagementKeyOK(t *testing.T) { /* mock HTTPDo 200 with data; GET models; assert 200 + ids */ }
func TestModelsEndpointFallbackOnUpstream401(t *testing.T) { /* both upstream fail; assert 200 + warning + gpt-6-luna */ }
```

Use Request headers: `Authorization: Bearer mgmt-key`, `Host: cpa.test`.

- [ ] **Step 2: RED** — `docker compose exec -T dev go test ./internal/management/ -run TestModelsEndpoint -count=1`

- [ ] **Step 3: Implement** route wiring only (logic already in Task 3).

- [ ] **Step 4: GREEN** — same command.

- [ ] **Step 5:** Do **not** commit.

---

### Task 5: Management UI `loadModels()` → plugin endpoint

**Files:**
- Modify: `internal/management/ui.go` (JS `loadModels`)
- Modify: `internal/management/ui_test.go`

**Produces:**

- Const or inline URL: `/v0/management/plugins/codex-selective-ping/models` (same pattern as `STATUS_URL` / `RUN_URL`).
- `fetch(MODELS_URL, { headers: { Authorization: 'Bearer '+k } })` — **no** `fetch('/v1/models'…)`.
- On JSON: rebuild select from `data[].id`; if `warning` string non-empty, `setModelSelectHint(warning)` (or prefixed short message); on clean success clear hint.
- Keep orphan current value behavior via existing `rebuildModelSelect`.
- Client-side `keepModel` filter may remain as defense-in-depth **or** be simplified since server already filters — **locked:** keep client `keepModel` for now (YAGNI to delete); server still filters.

- [ ] **Step 1: Failing test** — change existing assertion that requires `'/v1/models'` to require the plugin path instead:

```go
if !strings.Contains(html, `/v0/management/plugins/codex-selective-ping/models`) {
	t.Fatal("JS must fetch plugin models endpoint")
}
if strings.Contains(html, `fetch('/v1/models'`) || strings.Contains(html, `fetch("/v1/models"`) {
	t.Fatal("JS must not call /v1/models directly")
}
```

- [ ] **Step 2: RED** — `docker compose exec -T dev go test ./internal/management/ -run TestRender|Model -count=1` (use the exact existing test name that currently requires `/v1/models`).

- [ ] **Step 3: Implement** JS change in `ui.go`.

- [ ] **Step 4: GREEN** — same test; fix any other UI tests that still expect `/v1/models`.

- [ ] **Step 5:** Do **not** commit.

---

### Task 6: README + full suite

**Files:**
- Modify: `README.md`, `README.zh-Hant.md`, `README.ja.md`

**Produces:** Document:

- Builtin default / empty `model` → `gpt-6-luna`
- Model dropdown options come from plugin `…/models` (proxied), not browser `GET /v1/models`

- [ ] **Step 1:** Update the three READMEs (search `gpt-5.6-luna` and any “Fixed model” / `/v1/models` wording).

- [ ] **Step 2: Full suite** — `docker compose exec -T dev go test ./... -count=1`  
  Expected: PASS.

- [ ] **Step 3:** Do **not** commit.

---

### Task 7: Code review + chat summary (no commit)

- [ ] **Step 1:** Re-read diff vs spec; fix Critical/Important issues.
- [ ] **Step 2:** Chat 正體中文摘要：行為變更、驗證指令與結果、review、尚未 commit 的檔案、是否開 issue／PR／bump／release。
- [ ] **Step 3:** Ask user before any `git commit` / push / PR / release.

---

## Self-review (plan vs spec)

| Spec requirement | Task |
| --- | --- |
| Plugin `GET …/models` | Task 4 |
| Same-origin from forwarded Host | Task 2–3 |
| Auth: Management Key → Codex token | Task 3 |
| `modelfilter` on server | Task 3 |
| 200 + fallback + warning | Task 3–4 |
| UI stops calling `/v1/models` | Task 5 |
| Default `gpt-6-luna` | Task 1 + 6 |
| No API key field / no ping transport change | Global constraints |
| No mid-flow commit | Every Task Step 5 / Task 7 |

No TBD placeholders. Types `modelsListResponse` / `fetchFilteredModels` / `originFromHeaders` shared across Tasks 2–4.
