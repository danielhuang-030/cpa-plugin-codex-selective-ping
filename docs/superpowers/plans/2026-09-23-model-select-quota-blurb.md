# Remove quota blurb + selectable ping model — implementation plan

> **For agentic workers:** Use TDD task-by-task. Steps use checkbox (`- [ ]`) syntax.  
> **Do not start TDD until the user confirms this plan in chat.**

**Goal:** Drop the obsolete「額度欄」help row; let users pick the ping model from CPA `GET /v1/models` (Codex/OpenAI/ChatGPT filtered) via a「此刻」`<select>`, persist as config `model`.

**Architecture:** Approach A — browser fetches `/v1/models`, filters client-side, saves `model` through existing `saveCfg()` → `PATCH .../config`. Pinger/runner use `config.EffectiveModel(cfg)`. Shared filter heuristics live in `internal/modelfilter` (Go unit tests); JS mirrors the same rules (comment: keep in sync).

**Tech stack:** Go 1.23, existing management UI (HTML/JS in `ui.go`), Docker Compose `dev` for tests.

**Spec:** `docs/superpowers/specs/2026-09-23-model-select-quota-blurb-design.md`  
**Issue:** [#21](https://github.com/danielhuang-030/cpa-plugin-codex-selective-ping/issues/21)

## Global constraints

- Chat / summaries: 正體中文
- Tests: `sudo docker compose exec -T dev go test ./… -count=1` (or package-scoped)
- No Cloud Agent
- **No git commit / push / PR / release until end-of-flow user confirmation** (issue 流程收尾再問)
- Empty `model` → `pinger.ModelName` (`gpt-5.6-luna`)
- Do not restore quota columns on account cards

---

### Task 1: Config `model` + EffectiveModel

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Produces: `Config.Model string` with `json:"model,omitempty"`; `func EffectiveModel(cfg Config) string`

- [ ] **Step 1: Failing tests**

```go
func TestParseModel(t *testing.T) {
	cfg, err := Parse(`{"schedule_enabled":true,"timezone":"Asia/Taipei","times":["06:00"],"model":"gpt-5"}`)
	if err != nil { t.Fatal(err) }
	if cfg.Model != "gpt-5" { t.Fatalf("model=%q", cfg.Model) }
}

func TestEffectiveModelEmptyFallsBack(t *testing.T) {
	if got := EffectiveModel(Config{}); got != "gpt-5.6-luna" { // pinger.ModelName — import or duplicate literal in test via pinger.ModelName
		t.Fatalf("got %q", got)
	}
}

func TestEffectiveModelUsesConfig(t *testing.T) {
	if got := EffectiveModel(Config{Model: "gpt-5"}); got != "gpt-5" {
		t.Fatalf("got %q", got)
	}
}
```

Prefer importing `pinger.ModelName` in the empty-fallback test to avoid drift.

- [ ] **Step 2: RED** — `sudo docker compose exec -T dev go test ./internal/config -count=1`
- [ ] **Step 3: Implement** — add `Model` to struct + JSON/YAML parse branches; `EffectiveModel` returns `strings.TrimSpace(cfg.Model)` or `pinger.ModelName` (config may import pinger, or keep fallback string in config and assert equal to `pinger.ModelName` in test — prefer `EffectiveModel` in `config` returning trimmed value and a separate constant only if import cycle; if cycle, put `EffectiveModel` in a tiny helper used by management/runner, or pass fallback into `EffectiveModel(cfg, fallback string)`).

**Import-cycle rule:** If `config` → `pinger` is unwanted, implement:

```go
func EffectiveModel(cfg Config, fallback string) string {
	if m := strings.TrimSpace(cfg.Model); m != "" {
		return m
	}
	return fallback
}
```

Call sites pass `pinger.ModelName`.

- [ ] **Step 4: GREEN** — same test command
- [ ] **Step 5:** Do **not** commit yet

---

### Task 2: modelfilter.Keep

**Files:**
- Create: `internal/modelfilter/filter.go`
- Create: `internal/modelfilter/filter_test.go`

**Interfaces:**
- Produces: `func Keep(id, ownedBy string) bool`

- [ ] **Step 1: Failing tests**

```go
func TestKeepCodexOpenAIChatGPT(t *testing.T) {
	cases := []struct{ id, owned string; want bool }{
		{"gpt-5.6-luna", "openai", true},
		{"codex-mini", "", true},
		{"chatgpt-4o", "openai", true},
		{"claude-sonnet", "anthropic", false},
		{"gemini-pro", "google", false},
		{"GPT-4o", "OpenAI", true},
	}
	for _, tc := range cases {
		if got := Keep(tc.id, tc.owned); got != tc.want {
			t.Fatalf("%s/%s: got %v want %v", tc.id, tc.owned, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: RED**
- [ ] **Step 3: Implement** — case-insensitive substring match on `id` or `ownedBy` for `codex`, `openai`, `chatgpt`, `gpt-`
- [ ] **Step 4: GREEN**
- [ ] **Step 5:** No commit

---

### Task 3: Pinger uses injectable model

**Files:**
- Modify: `internal/pinger/pinger.go`
- Modify: `internal/pinger/pinger_test.go`
- Modify: `internal/runner/limited_retry.go` (and tests)
- Modify: `internal/runner/runner.go`

**Interfaces:**
- Change: `PingAccount(..., model string)` and `pingOnce(..., model string)`
- Change: `pingFn` to include `model string`
- Runner passes `config.EffectiveModel(cfg, pinger.ModelName)`

- [ ] **Step 1: Failing test** — in `pinger_test.go`, drive a ping with a custom model (mock host) and assert JSON body `"model"` equals that value; empty/omitted path still uses `ModelName` if call site passes fallback.

- [ ] **Step 2: RED** — `sudo docker compose exec -T dev go test ./internal/pinger ./internal/runner -count=1`
- [ ] **Step 3: Minimal wiring** — thread `model` through `PingAccount` → `pingOnce` → `codexBody.Model`; update `pingWithLimitedRetry` + `Runner` to pass effective model
- [ ] **Step 4: GREEN**
- [ ] **Step 5:** No commit

---

### Task 4: Status shows effective model

**Files:**
- Modify: `internal/management/handler.go` (status `Model: config.EffectiveModel(cfg, pinger.ModelName)`)
- Modify: tests under `internal/management` if status asserts model

- [ ] **Step 1: Failing test** — status/handler test with cfg.Model set → JSON/HTML shows that model
- [ ] **Step 2: RED**
- [ ] **Step 3: Implement**
- [ ] **Step 4: GREEN**
- [ ] **Step 5:** No commit

---

### Task 5: Remove quota help row

**Files:**
- Modify: `internal/management/ui.go` (HTML + sprintf args)
- Modify: `internal/management/i18n.go` (optional: delete unused keys)
- Modify: `internal/management/i18n_test.go` (drop `how_quota_removed` / `how_quota_removed_v` from required keys)
- Modify: any UI test asserting those keys/rows

- [ ] **Step 1: Failing test** — rendered HTML for principles/how block must **not** contain `how_quota_removed` / `data-i18n="how_quota_removed"`
- [ ] **Step 2: RED**
- [ ] **Step 3: Remove the metric row + args; clean i18n keys if unused**
- [ ] **Step 4: GREEN**
- [ ] **Step 5:** No commit

---

### Task 6:「此刻」model `<select>` + JS fetch/filter/save

**Files:**
- Modify: `internal/management/ui.go` (rail markup + CSS if needed + JS)
- Modify: `internal/management/ui_test.go`

**Behavior to implement in JS:**
1. Replace rail model `<span class="v">…</span>` with  
   `<select id="model-select" data-testid="model-select">` pre-populated with current effective model option.
2. On load: `fetch('/v1/models', { headers: { Authorization: 'Bearer '+key() } })` (same key as management; if 401, try without changing config — show hint in `#result` or a small rail hint).
3. Parse OpenAI-style `{ data: [ { id, owned_by } ] }`; keep entries where heuristics match Task 2 (mirror `modelfilter.Keep`; comment `// sync with internal/modelfilter.Keep`).
4. Rebuild `<select>` options; if current value missing, prepend it.
5. On `change`: set in-memory value; include `model` in `saveCfg()` body:  
   `body.model = document.getElementById('model-select').value.trim()`
6. Fetch failure: leave single current option; non-blocking error text; do not clear config.

- [ ] **Step 1: Failing UI tests**
  - Rendered HTML contains `data-testid="model-select"` (or `id="model-select"`)
  - Inline JS contains `'/v1/models'` and a `keepModel` (or equivalent) function mentioning `codex` / `openai` / `chatgpt` / `gpt-`
  - `saveCfg` body includes `model`
- [ ] **Step 2: RED**
- [ ] **Step 3: Implement markup + JS**
- [ ] **Step 4: GREEN**
- [ ] **Step 5:** No commit

---

### Task 7: README + full suite

**Files:**
- Modify: `README.md` / `README.zh-Hant.md` / `README.ja.md` — document `model` (optional; empty → `gpt-5.6-luna`); remove “Fixed model” wording if present

- [ ] Update config tables
- [ ] `sudo docker compose exec -T dev go test ./... -count=1`
- [ ] 正體中文 summary + code review notes
- [ ] **Ask user** before commit / PR / bump / release (issue #21 流程)

---

## Non-negotiables

- No TDD before plan confirmation
- No commit until user confirms after summary
- Approach A only unless `/v1/models` from the browser is proven impossible — then stop and ask before switching to proxy (Approach B)
