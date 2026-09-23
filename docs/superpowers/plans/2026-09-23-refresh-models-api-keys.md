# Refresh models via Management Key → api-keys — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.  
> **Do not start TDD until the user confirms this plan in chat.**  
> **sp-flow:** No `git commit` / push / PR / release until end-of-flow user confirmation. Chat in 正體中文. No Cloud Agent. Tests via Docker Compose `dev`.

**Goal:** Let the Management UI refresh the model `<select>` with a real CPA model list by chaining the page Management Key through `GET /v0/management/api-keys` to a proxy API key for `GET /v1/models`, behind an explicit **更新模型** button (auto-load only when session already has a key).

**Architecture:** Keep the existing plugin `GET …/models` proxy. Extend `fetchFilteredModels` credential cascade to (1) Management Key → `/v0/management/api-keys` → first proxy API key → `/v1/models`, (2) Codex Access Token → `/v1/models`, (3) HTTP 200 + fallback ids + `warning`. UI adds a compact refresh button beside `#model-select`; empty key on button click mirrors save (`msgNeedKey` in `#result`).

**Tech Stack:** Go (`internal/management`, `internal/hostapi`, `internal/modelfilter`, `internal/pinger`, `internal/selector`), inline Management UI JS/CSS/i18n, Docker Compose service `dev`.

**Spec:** `docs/superpowers/specs/2026-09-23-refresh-models-api-keys-design.md`

## Global Constraints

- Chat / progress: 正體中文
- Tests: `docker compose exec -T dev go test ./… -count=1` (narrow `-run` while iterating)
- No Cloud Agent
- **No git commit / push / PR / release until user confirms after review**
- Do not add a user-supplied API key field
- Do not change ping transport (still Codex URL + AccessToken)
- Do not change default `pinger.ModelName` (`gpt-6-luna`)
- Upstream order locked: **api-keys → Codex token → fallback** (Management Key is **not** sent directly to `/v1/models`)
- Plugin models route path unchanged: `/v0/management/plugins/codex-selective-ping/models`
- Secrets never appear in models JSON responses

## File map

| File | Role |
| --- | --- |
| `internal/management/models.go` | Add `firstAPIKeyFromBody`; rewrite `fetchFilteredModels` cascade |
| `internal/management/models_test.go` | Parser + cascade tests; update obsolete “mgmt key hits /v1/models” cases |
| `internal/management/i18n.go` | Add `refresh_models` (zh-Hant / en / ja) |
| `internal/management/i18n_test.go` | Assert key present in all locales (if pattern exists; else extend) |
| `internal/management/ui.go` | Button + CSS; `refreshModels()`; wire i18n; busy disable |
| `internal/management/ui_test.go` | Assert button, i18n key, `refreshModels` / need-key behavior strings |
| `README.md` / `README.zh-Hant.md` / `README.ja.md` | Short note: refresh uses Management Key → api-keys |

---

### Task 1: Parse `api-keys` JSON

**Files:**
- Modify: `internal/management/models.go`
- Modify: `internal/management/models_test.go`

**Produces:** `firstAPIKeyFromBody(body []byte) (string, error)` returning the first non-empty proxy API key.

- [ ] **Step 1: Failing tests** — add to `models_test.go`:

```go
func TestFirstAPIKeyFromBodyStringArray(t *testing.T) {
	got, err := firstAPIKeyFromBody([]byte(`{"api-keys":[" sk-a ","sk-b"]}`))
	if err != nil || got != "sk-a" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestFirstAPIKeyFromBodyObjectArray(t *testing.T) {
	got, err := firstAPIKeyFromBody([]byte(`{"api-keys":[{"api-key":"sk-obj"},{"key":"sk-2"}]}`))
	if err != nil || got != "sk-obj" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestFirstAPIKeyFromBodyEmpty(t *testing.T) {
	if _, err := firstAPIKeyFromBody([]byte(`{"api-keys":[]}`)); err == nil {
		t.Fatal("expected error")
	}
	if _, err := firstAPIKeyFromBody([]byte(`{}`)); err == nil {
		t.Fatal("expected error")
	}
}
```

Also accept object field aliases `apiKey` and `key` (covered by object test / add one-liner if needed).

- [ ] **Step 2: RED**

```bash
docker compose exec -T dev go test ./internal/management/ -run 'TestFirstAPIKeyFromBody' -count=1
```

Expected: FAIL (`firstAPIKeyFromBody` undefined).

- [ ] **Step 3: Minimal implementation** in `models.go`:

```go
func firstAPIKeyFromBody(body []byte) (string, error) {
	var root struct {
		APIKeys []json.RawMessage `json:"api-keys"`
	}
	if err := json.Unmarshal(body, &root); err != nil {
		return "", err
	}
	for _, raw := range root.APIKeys {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			s = strings.TrimSpace(s)
			if s != "" {
				return s, nil
			}
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil {
			continue
		}
		for _, k := range []string{"api-key", "apiKey", "key"} {
			if v, ok := obj[k]; ok {
				if s, ok := v.(string); ok {
					s = strings.TrimSpace(s)
					if s != "" {
						return s, nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("no api key")
}
```

- [ ] **Step 4: GREEN** — same `-run 'TestFirstAPIKeyFromBody'`.

- [ ] **Step 5:** Do **not** commit.

---

### Task 2: Cascade — api-keys → Codex → fallback

**Files:**
- Modify: `internal/management/models.go` (`fetchFilteredModels`)
- Modify: `internal/management/models_test.go` (replace/repurpose old management-key→`/v1/models` success test)

**Produces:** New auth order per spec; existing Codex / fallback behaviors preserved.

- [ ] **Step 1: Failing / updated tests**

Replace `TestFetchFilteredModelsManagementKeyOK` with api-key success (mgmt key must **not** be used as Bearer on `/v1/models`):

```go
func TestFetchFilteredModelsAPIKeyOK(t *testing.T) {
	keysBody, _ := json.Marshal(map[string]any{"api-keys": []string{"sk-proxy"}})
	modelsBody, _ := json.Marshal(map[string]any{
		"data": []map[string]any{
			{"id": "gpt-6-luna", "owned_by": "openai"},
			{"id": "claude-3", "owned_by": "anthropic"},
		},
	})
	h := &modelsHost{script: []hostapi.HTTPResponse{
		{StatusCode: 200, Body: keysBody},
		{StatusCode: 200, Body: modelsBody},
	}}
	out := fetchFilteredModels(context.Background(), h, "http://cpa.test", "mgmt-key", "")
	if out.Warning != "" {
		t.Fatalf("warning=%q", out.Warning)
	}
	if len(h.calls) != 2 {
		t.Fatalf("calls=%d", len(h.calls))
	}
	if h.calls[0].URL != "http://cpa.test/v0/management/api-keys" {
		t.Fatalf("keys url=%q", h.calls[0].URL)
	}
	if firstHeader(h.calls[0].Headers, "Authorization") != "Bearer mgmt-key" {
		t.Fatalf("keys auth=%q", firstHeader(h.calls[0].Headers, "Authorization"))
	}
	if h.calls[1].URL != "http://cpa.test/v1/models" {
		t.Fatalf("models url=%q", h.calls[1].URL)
	}
	if firstHeader(h.calls[1].Headers, "Authorization") != "Bearer sk-proxy" {
		t.Fatalf("models auth=%q", firstHeader(h.calls[1].Headers, "Authorization"))
	}
	ids := modelIDs(out)
	if !hasID(ids, "gpt-6-luna") || hasID(ids, "claude-3") {
		t.Fatalf("ids=%v", ids)
	}
}
```

Add: api-keys fail then Codex OK (two failing early responses then models OK — script length must match call order: api-keys 401, then optionally skipped models with api key, then codex models). Preferred script:

```go
func TestFetchFilteredModelsAPIKeysFailThenCodexOK(t *testing.T) {
	okBody, _ := json.Marshal(map[string]any{
		"data": []map[string]any{{"id": "gpt-4o", "owned_by": "openai"}},
	})
	h := &modelsHost{
		files: []hostapi.AuthFile{{AuthIndex: "1", Provider: "codex", Name: "a"}},
		creds: map[string][]byte{"1": mustJSON(map[string]any{"access_token": "codex-tok"})},
		script: []hostapi.HTTPResponse{
			{StatusCode: 401, Body: []byte(`{"error":"no"}`)}, // api-keys
			{StatusCode: 200, Body: okBody},                   // /v1/models with codex
		},
	}
	out := fetchFilteredModels(context.Background(), h, "https://cpa.test", "mgmt-key", "kept-orphan")
	if out.Warning != "" {
		t.Fatalf("warning=%q", out.Warning)
	}
	if len(h.calls) != 2 {
		t.Fatalf("calls=%d urls=%v", len(h.calls), []string{h.calls[0].URL, h.calls[1].URL})
	}
	if firstHeader(h.calls[1].Headers, "Authorization") != "Bearer codex-tok" {
		t.Fatalf("second auth=%q", firstHeader(h.calls[1].Headers, "Authorization"))
	}
	if !hasID(modelIDs(out), "gpt-4o") {
		t.Fatalf("ids=%v", modelIDs(out))
	}
}
```

Update `TestFetchFilteredModelsBothFailFallback` so the script includes api-keys failure then Codex `/v1/models` failure (still fallback with configured model + `pinger.ModelName`).

Keep / adjust `TestBearerFromAuthHeader` unchanged.

- [ ] **Step 2: RED** — run new/updated tests; expect FAIL on URL/auth ordering until implementation changes.

```bash
docker compose exec -T dev go test ./internal/management/ -run 'TestFetchFilteredModels' -count=1
```

- [ ] **Step 3: Implement cascade** in `fetchFilteredModels`:

Pseudo-order (keep `try(token,label)` helper for `/v1/models` only):

1. If `managementKey != ""` and host/origin set: `HTTPDo` GET `origin+"/v0/management/api-keys"` with `Authorization: Bearer `+managementKey. On 2xx, `firstAPIKeyFromBody`; if key found, `try(apiKey, "api-key")` and return on success. On any failure, set `lastWarn` accordingly (do not abort cascade).
2. `firstCodexAccessToken` → `try(tok, "codex")`.
3. Return `fallbackModelItems(cfgModel)` + `fallbackWarning(lastWarn)`.

Do **not** call `try(managementKey, "management")`.

- [ ] **Step 4: GREEN**

```bash
docker compose exec -T dev go test ./internal/management/ -run 'TestFetchFilteredModels|TestFirstAPIKeyFromBody|TestBearerFromAuthHeader' -count=1
```

- [ ] **Step 5:** Do **not** commit.

---

### Task 3: i18n `refresh_models`

**Files:**
- Modify: `internal/management/i18n.go`
- Modify: `internal/management/i18n_test.go` and/or `ui_test.go` (follow existing “key present in all maps” pattern)

**Produces:**

| locale | value |
| --- | --- |
| zh-Hant | `更新模型` |
| en | `Refresh models` |
| ja | `モデル更新` |

- [ ] **Step 1: Failing test** — assert `t("refresh_models")` / map lookup non-empty for zh-Hant, en, ja (mirror neighboring key tests).

- [ ] **Step 2: RED**

```bash
docker compose exec -T dev go test ./internal/management/ -run 'RefreshModels|refresh_models|I18n' -count=1
```

- [ ] **Step 3: Add keys** to all three locale maps in `i18n.go` near `rail_model` / action labels.

- [ ] **Step 4: GREEN** — same test command.

- [ ] **Step 5:** Do **not** commit.

---

### Task 4: UI — button + refresh path

**Files:**
- Modify: `internal/management/ui.go`
- Modify: `internal/management/ui_test.go`

**Produces:** Button to the right of `#model-select`; `refreshModels()`; busy disable; empty-key → `#result` + `msgNeedKey`; boot still auto-`loadModels()` when `key()` non-empty.

- [ ] **Step 1: Failing UI tests** — extend `ui_test.go` to require in rendered HTML/JS:

  - `data-i18n="refresh_models"` (or visible `更新模型` for zh-Hant render)
  - `onclick="refreshModels()"` (or equivalent named handler)
  - `id="refresh-models"` or `data-testid="refresh-models"` on the button
  - JS contains `function refreshModels` and uses `msgNeedKey` when key empty
  - Still contains `MODELS_URL = '/v0/management/plugins/codex-selective-ping/models'`
  - Must **not** reintroduce browser calls to bare `'/v1/models'`

Example assertions (adapt to existing helper that renders status HTML):

```go
func TestRenderStatusPageRefreshModelsButton(t *testing.T) {
	html := renderStatusPageForTest(t) // use the suite's existing render helper/name
	for _, want := range []string{
		`data-testid="refresh-models"`,
		`refreshModels(`,
		`data-i18n="refresh_models"`,
		`/v0/management/plugins/codex-selective-ping/models`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q", want)
		}
	}
	if strings.Contains(html, `fetch('/v1/models`) || strings.Contains(html, `fetch("/v1/models`) {
		t.Fatal("UI must not call /v1/models directly")
	}
}
```

- [ ] **Step 2: RED**

```bash
docker compose exec -T dev go test ./internal/management/ -run 'TestRenderStatusPageRefreshModelsButton' -count=1
```

- [ ] **Step 3: Implement UI**

HTML (inside `.metric.metric-model`, after `<select id="model-select"…>`):

```html
<button type="button" class="btn-secondary btn-compact" id="refresh-models" data-testid="refresh-models" onclick="refreshModels()" data-i18n="refresh_models">%s</button>
```

Pass `html.EscapeString(t("refresh_models"))` into the sprintf slot.

CSS (near `#model-select`):

```css
.metric.metric-model #refresh-models { flex: 0 0 auto; padding: 6px 10px; font-size: 12px; }
.metric.metric-model { flex-wrap: wrap; }
```

JS:

```javascript
async function refreshModels(){
  const k=key();
  if(!k){
    const o=document.getElementById('result');
    if(o) o.textContent=msgNeedKey;
    return;
  }
  const btn=document.getElementById('refresh-models');
  if(btn){ btn.disabled=true; }
  try{ await loadModels(); }
  finally{ if(btn){ btn.disabled=false; } }
}
```

Keep `loadModels()` early-return on empty key **silent** (boot path). Only `refreshModels()` shows `msgNeedKey`.

Ensure boot still calls `loadModels()` when session key exists (existing call site).

- [ ] **Step 4: GREEN**

```bash
docker compose exec -T dev go test ./internal/management/ -run 'TestRenderStatusPageRefreshModelsButton|TestRenderStatusPage' -count=1
```

- [ ] **Step 5:** Do **not** commit.

---

### Task 5: README note + full suite

**Files:**
- Modify: `README.md`, `README.zh-Hant.md`, `README.ja.md` (only sections that describe the model select / Management Key)
- Verify: entire module tests

- [ ] **Step 1:** Add one short sentence near model-select docs: refreshing the list uses the Management Key to read CPA `api-keys`, then lists `/v1/models` through the plugin proxy; empty key prompts like save.

- [ ] **Step 2: Full GREEN**

```bash
docker compose exec -T dev go test ./... -count=1
```

Expected: all pass.

- [ ] **Step 3:** Do **not** commit.

---

### Task 6: Self code review + chat summary (no commit)

- [ ] Re-read diff against spec checklist:
  - Button placement + empty-key = save behavior
  - Session key auto-load retained
  - Cascade api-keys → Codex → fallback
  - No secrets in response
  - Default model unchanged
- [ ] **SendToUser** 正體中文摘要：行為變更、驗證指令與結果、Critical/Important/Minor、`git status` 未 commit 清單
- [ ] Ask whether to commit / push / PR / release — **stop**

---

## Spec coverage checklist

| Spec item | Task |
| --- | --- |
| `firstAPIKeyFromBody` / api-keys parse | Task 1 |
| Cascade api-keys → Codex → fallback | Task 2 |
| No direct Management Key on `/v1/models` | Task 2 |
| i18n `refresh_models` | Task 3 |
| Button right of select; need-key on click; boot silent without key; auto-load with session key | Task 4 |
| README | Task 5 |
| No commit until user OK | Tasks 1–6 Step “Do not commit” + Task 6 ask |

## Placeholder / consistency self-check

- Function names match code: `fetchFilteredModels`, `loadModels`, `key`, `msgNeedKey`, `pinger.ModelName`
- DOM ids: `#model-select`, `#model-select-hint`, `#result`, `#refresh-models`
- Paths: `/v0/management/api-keys`, `/v1/models`, plugin `…/models`
- Fallback id: `gpt-6-luna` via `pinger.ModelName`
