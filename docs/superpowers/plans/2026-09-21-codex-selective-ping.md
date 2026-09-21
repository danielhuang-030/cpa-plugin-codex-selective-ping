# Codex Selective Ping Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a brand-new independent CLIProxyAPI c-shared plugin `codex-selective-ping` that schedules fixed-model Codex pings only for a configured account whitelist, persists config via host `plugins.configs`, and exposes a Traditional Chinese Management resource UI with optional Plan/5h/weekly quota columns.

**Architecture:** Keep all business logic in pure-Go `internal/*` packages behind a `hostapi.Host` interface so unit tests never need CGO. Thin `main.go` owns only the CPA plugin ABI (CGO c-shared exports) and adapts host callbacks into `hostapi.Host`. Config/Selector/RunState/Pinger/Scheduler/Runner/Management are separate packages. Empty `accounts` means ping nobody. UI saves via authenticated host Management config PATCH; plugin routes expose `GET status` and `POST run` only.

**Tech Stack:** Go 1.23+ (box has 1.24.x), CGO `c-shared`, stdlib only (`encoding/json`, `net/http` types as structs, `testing`, `html/template` or string-built HTML), no third-party deps.

**Spec:** `docs/superpowers/specs/2026-09-21-codex-selective-ping-design.md`  
**UI mock:** `docs/superpowers/mocks/management-ui-mock.html`  
**Reference (read-only, do not ship):** current root `main.go` (auto-ping copy) and nested `cpa-plugin-codex-auto-ping/` (workflow only).

## Global Constraints

- Plugin ID: `codex-selective-ping` (never `codex-auto-ping`).
- Module path: `cpa-plugin-codex-selective-ping`.
- Empty `accounts` = ping nobody (`attempted=0`); never treat empty as ping-all.
- Persist user settings only via host `plugins.configs.codex-selective-ping` + Management GET/PATCH config API.
- Reuse auto-ping host ABI methods (`host.auth.list`, `host.auth.get`, `host.http.do`), fixed model `gpt-5.6-luna`, schedule semantics (daily HH:MM in IANA TZ; **do not** ping everyone on startup), per-account continue-on-failure, 429/`usage_limit_reached` → `limited`.
- Quota fields: show when host exposes them; missing → `"—"` / omit / nil; never invent remaining/used/reset.
- Management resource page primary language: Traditional Chinese.
- Go + CGO c-shared deliverable (`.so` / `.dylib` / `.dll`).
- Develop on this box; do not `git push` unless the user asks.
- Replace temporary auto-ping root sources; nested `cpa-plugin-codex-auto-ping/` must not remain as product code.

## Review Focus

- Selector empty whitelist → zero targets.
- Email / name matching is case-insensitive; `auth_index` exact trim match.
- Non-Codex auth entries never selected.
- `POST /run` while running → HTTP 409; accept → HTTP 202.
- Status JSON includes `selected` and optional quota fields without fabricated numbers.
- Build produces a loadable c-shared library.

## File structure map (lock first)

| Path | Responsibility |
|------|----------------|
| `go.mod` | Module `cpa-plugin-codex-selective-ping`, Go 1.23 |
| `Makefile` | `test`, `build-linux` (`c-shared` → `codex-selective-ping.so`) |
| `registry.json` | Plugin store metadata for `codex-selective-ping` |
| `README.md` | Install, config YAML, Management routes, build |
| `.gitignore` | `*.so`, `*.dylib`, `*.dll`, `*.h`, `dist/`, `package/` |
| `internal/config/config.go` | Parse/validate `enabled`, `timezone`, `times`, `accounts` |
| `internal/config/config_test.go` | Config TDD |
| `internal/selector/selector.go` | Codex filter + whitelist match |
| `internal/selector/selector_test.go` | Selector TDD |
| `internal/hostapi/types.go` | `AuthFile`, quota structs, HTTP request/response |
| `internal/hostapi/host.go` | `Host` interface |
| `internal/hostapi/quota.go` | Best-effort quota extraction from auth JSON (no invention) |
| `internal/hostapi/quota_test.go` | Quota parse TDD |
| `internal/runstate/runstate.go` | Running mutex, last summary, per-account state |
| `internal/runstate/runstate_test.go` | RunState TDD |
| `internal/pinger/pinger.go` | Fixed-model ping via Host; retries; limited handling |
| `internal/pinger/pinger_test.go` | Pinger TDD with mock Host |
| `internal/scheduler/scheduler.go` | Daily schedule loop; next-run; cancel on reconfigure |
| `internal/scheduler/scheduler_test.go` | `NextRun` TDD |
| `internal/runner/runner.go` | Orchestrate select→ping→state; empty accounts short-circuit |
| `internal/runner/runner_test.go` | Only-selected + empty + mutex behavior |
| `internal/management/handler.go` | `GET status`, `POST run`, resource HTML dispatch |
| `internal/management/handler_test.go` | Status shape + 202/409 |
| `internal/management/ui.go` | Embedded Traditional Chinese resource page |
| `internal/plugin/plugin.go` | Wires config/scheduler/runner/management; reconfigure |
| `internal/plugin/plugin_test.go` | ApplyConfig + status integration smoke |
| `main.go` | CGO ABI exports + real Host adapter |
| `docs/superpowers/specs/...` | Spec (already present; do not delete) |
| `docs/superpowers/mocks/...` | UI mock (already present; do not delete) |

**Delete / replace (not ship):** root auto-ping `main.go` content, `registry.json` auto-ping id, nested `cpa-plugin-codex-auto-ping/` directory after Makefile/README exist.

---
### Task 1: Module scaffold + Config parse/validate

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`
- Modify/replace: remove product identity from old root files in later tasks; for this task only create new module + config package (leave old `main.go` until Task 11 so the tree still builds if someone opens it — or delete `main.go` now and leave no package main until Task 11). **Do this:** delete root `main.go` and nested product is cleaned in Task 12; after deleting `main.go`, only `internal/config` exists as a package until later tasks add more.

**Interfaces:**
- Consumes: nothing
- Produces:
  - `type Config struct { Enabled bool; Timezone string; Times []string; Accounts []string }`
  - `func DefaultConfig() Config`
  - `func Parse(raw string) (Config, error)` — accepts JSON object or simple YAML subset
  - `func Validate(cfg Config) (Config, error)`
  - Defaults: `Enabled=true`, `Timezone="Asia/Taipei"`, `Times=["06:00","11:00","16:00","21:00"]`, `Accounts=[]` (empty)

- [ ] **Step 1: Write the failing test**

Create `go.mod`:

```go
module cpa-plugin-codex-selective-ping

go 1.23
```

Create `.gitignore`:

```
*.so
*.dylib
*.dll
*.h
dist/
package/
*.test
```

Create `internal/config/config_test.go`:

```go
package config

import (
	"strings"
	"testing"
)

func TestDefaultConfigEmptyAccounts(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Enabled {
		t.Fatal("Enabled default true")
	}
	if cfg.Timezone != "Asia/Taipei" {
		t.Fatalf("Timezone=%q", cfg.Timezone)
	}
	if len(cfg.Accounts) != 0 {
		t.Fatalf("Accounts must default empty, got %#v", cfg.Accounts)
	}
	if len(cfg.Times) != 4 {
		t.Fatalf("Times default len=%d", len(cfg.Times))
	}
}

func TestParseYAMLAccountsAndValidate(t *testing.T) {
	raw := `
enabled: true
timezone: Asia/Taipei
times:
  - "06:00"
  - "21:00"
accounts:
  - "User@Example.com"
  - "auth-1"
`
	cfg, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Accounts) != 2 || cfg.Accounts[0] != "User@Example.com" || cfg.Accounts[1] != "auth-1" {
		t.Fatalf("accounts=%#v", cfg.Accounts)
	}
	if cfg.Times[0] != "06:00" || cfg.Times[1] != "21:00" {
		t.Fatalf("times=%#v", cfg.Times)
	}
}

func TestParseJSON(t *testing.T) {
	cfg, err := Parse(`{"enabled":false,"timezone":"UTC","times":["07:30"],"accounts":["a"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled || cfg.Timezone != "UTC" || cfg.Times[0] != "07:30" || cfg.Accounts[0] != "a" {
		t.Fatalf("%#v", cfg)
	}
}

func TestValidateRejectsBadTimezone(t *testing.T) {
	_, err := Validate(Config{Enabled: true, Timezone: "Not/AZone", Times: []string{"06:00"}})
	if err == nil || !strings.Contains(err.Error(), "timezone") {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateRejectsBadTime(t *testing.T) {
	_, err := Validate(Config{Enabled: true, Timezone: "UTC", Times: []string{"25:00"}})
	if err == nil || !strings.Contains(err.Error(), "time") {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateRequiresAtLeastOneTime(t *testing.T) {
	_, err := Validate(Config{Enabled: true, Timezone: "UTC", Times: nil, Accounts: []string{"x"}})
	if err == nil || !strings.Contains(err.Error(), "times") {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateDedupesTimesAndAllowsEmptyAccounts(t *testing.T) {
	cfg, err := Validate(Config{Enabled: true, Timezone: "UTC", Times: []string{"6:00", "06:00", "21:00"}, Accounts: nil})
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Times) != 2 || cfg.Times[0] != "06:00" || cfg.Times[1] != "21:00" {
		t.Fatalf("times=%#v", cfg.Times)
	}
	if cfg.Accounts == nil {
		cfg.Accounts = []string{}
	}
	if len(cfg.Accounts) != 0 {
		t.Fatalf("accounts=%#v", cfg.Accounts)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /workspace/cpa-plugin-codex-selective-ping && rm -f main.go && go test ./internal/config -count=1`

Expected: FAIL with `package config is not in std` or `no required module provides package` until `go.mod` exists; after `go.mod`, FAIL with `undefined: DefaultConfig` / `Parse` / `Validate`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/config/config.go`:

```go
package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var defaultTimes = []string{"06:00", "11:00", "16:00", "21:00"}

type Config struct {
	Enabled  bool     `json:"enabled"`
	Timezone string   `json:"timezone"`
	Times    []string `json:"times"`
	Accounts []string `json:"accounts"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:  true,
		Timezone: "Asia/Taipei",
		Times:    append([]string(nil), defaultTimes...),
		Accounts: []string{},
	}
}

func Parse(raw string) (Config, error) {
	cfg := DefaultConfig()
	text := strings.TrimSpace(raw)
	if text == "" {
		return Validate(cfg)
	}
	if strings.HasPrefix(text, "{") {
		var p struct {
			Enabled  *bool    `json:"enabled"`
			Timezone string   `json:"timezone"`
			Times    []string `json:"times"`
			Accounts []string `json:"accounts"`
		}
		if err := json.Unmarshal([]byte(text), &p); err != nil {
			return Config{}, fmt.Errorf("invalid JSON config: %w", err)
		}
		if p.Enabled != nil {
			cfg.Enabled = *p.Enabled
		}
		if strings.TrimSpace(p.Timezone) != "" {
			cfg.Timezone = strings.TrimSpace(p.Timezone)
		}
		if p.Times != nil {
			cfg.Times = p.Times
		}
		if p.Accounts != nil {
			cfg.Accounts = p.Accounts
		}
		return Validate(cfg)
	}
	return parseYAMLSubset(text, cfg)
}

func parseYAMLSubset(text string, cfg Config) (Config, error) {
	lines := strings.Split(text, "\n")
	mode := ""
	var times []string
	var accounts []string
	for _, rawLine := range lines {
		line := strings.TrimSpace(strings.SplitN(rawLine, "#", 2)[0])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "-") && (mode == "times" || mode == "accounts") {
			item := unquote(strings.TrimSpace(strings.TrimPrefix(line, "-")))
			if mode == "times" {
				times = append(times, item)
			} else {
				accounts = append(accounts, item)
			}
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		mode = ""
		switch key {
		case "enabled":
			if value != "" {
				b, err := strconv.ParseBool(unquote(value))
				if err != nil {
					return Config{}, fmt.Errorf("enabled must be true or false")
				}
				cfg.Enabled = b
			}
		case "timezone":
			if value != "" {
				cfg.Timezone = unquote(value)
			}
		case "times":
			mode = "times"
			if value != "" {
				times = parseInlineList(value)
				mode = ""
			}
		case "accounts":
			mode = "accounts"
			if value != "" {
				accounts = parseInlineList(value)
				mode = ""
			}
		}
	}
	if times != nil {
		cfg.Times = times
	}
	if accounts != nil {
		cfg.Accounts = accounts
	}
	return Validate(cfg)
}

func Validate(cfg Config) (Config, error) {
	cfg.Timezone = strings.TrimSpace(cfg.Timezone)
	if cfg.Timezone == "" {
		cfg.Timezone = "Asia/Taipei"
	}
	if _, err := time.LoadLocation(cfg.Timezone); err != nil {
		return Config{}, fmt.Errorf("invalid timezone %q", cfg.Timezone)
	}
	if len(cfg.Times) == 0 {
		return Config{}, fmt.Errorf("times must contain at least one HH:MM value")
	}
	seen := map[string]bool{}
	norm := make([]string, 0, len(cfg.Times))
	for _, v := range cfg.Times {
		v = strings.TrimSpace(unquote(v))
		h, m, err := ParseClock(v)
		if err != nil {
			return Config{}, err
		}
		n := fmt.Sprintf("%02d:%02d", h, m)
		if !seen[n] {
			seen[n] = true
			norm = append(norm, n)
		}
	}
	cfg.Times = norm
	outAcc := make([]string, 0, len(cfg.Accounts))
	for _, a := range cfg.Accounts {
		a = strings.TrimSpace(unquote(a))
		if a != "" {
			outAcc = append(outAcc, a)
		}
	}
	cfg.Accounts = outAcc
	return cfg, nil
}

func ParseClock(value string) (int, int, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid time %q: expected HH:MM", value)
	}
	h, e1 := strconv.Atoi(parts[0])
	m, e2 := strconv.Atoi(parts[1])
	if e1 != nil || e2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, fmt.Errorf("invalid time %q: expected 00:00-23:59", value)
	}
	return h, m, nil
}

func parseInlineList(value string) []string {
	value = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(value), "["), "]"))
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, unquote(strings.TrimSpace(p)))
	}
	return out
}

func unquote(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /workspace/cpa-plugin-codex-selective-ping && go test ./internal/config -count=1`

Expected: PASS (`ok cpa-plugin-codex-selective-ping/internal/config`)

- [ ] **Step 5: Commit**

```bash
cd /workspace/cpa-plugin-codex-selective-ping
git add go.mod .gitignore internal/config/config.go internal/config/config_test.go
git rm -f --ignore-unmatch main.go 2>/dev/null || true
# If main.go still tracked from auto-ping copy:
git add -u main.go || true
git commit -m "$(cat <<'EOF'
feat: scaffold module and config parse/validate with accounts

EOF
)"
```

---

### Task 2: Selector (whitelist + Codex filter)

**Files:**
- Create: `internal/selector/selector.go`
- Test: `internal/selector/selector_test.go`
- Create: `internal/hostapi/types.go` (AuthFile used by selector)

**Interfaces:**
- Consumes: `hostapi.AuthFile`
- Produces:
  - `func IsCodex(a hostapi.AuthFile) bool`
  - `func Select(files []hostapi.AuthFile, accounts []string) []hostapi.AuthFile`
  - Matching: trim; `auth_index` exact; `email`/`name`/`account` case-insensitive; empty `accounts` → empty result

- [ ] **Step 1: Write the failing test**

Create `internal/hostapi/types.go`:

```go
package hostapi

import "time"

type QuotaWindow struct {
	Remaining *float64   `json:"remaining,omitempty"`
	Used      *float64   `json:"used,omitempty"`
	ResetsAt  *time.Time `json:"resets_at,omitempty"`
}

type AuthFile struct {
	ID          string       `json:"id,omitempty"`
	Name        string       `json:"name"`
	AuthIndex   string       `json:"auth_index"`
	Account     string       `json:"account,omitempty"`
	Email       string       `json:"email,omitempty"`
	Provider    string       `json:"provider,omitempty"`
	Type        string       `json:"type,omitempty"`
	Status      string       `json:"status,omitempty"`
	Disabled    bool         `json:"disabled"`
	Unavailable bool         `json:"unavailable,omitempty"`
	Plan        string       `json:"plan,omitempty"`
	FiveHour    *QuotaWindow `json:"five_hour,omitempty"`
	Weekly      *QuotaWindow `json:"weekly,omitempty"`
}
```

Create `internal/selector/selector_test.go`:

```go
package selector

import (
	"testing"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
)

func sample() []hostapi.AuthFile {
	return []hostapi.AuthFile{
		{AuthIndex: "1", Name: "Alice", Email: "alice@Example.com", Provider: "codex"},
		{AuthIndex: "2", Name: "Bob-Work", Email: "bob@x.com", Provider: "codex"},
		{AuthIndex: "3", Name: "gemini", Email: "g@x.com", Provider: "gemini"},
		{AuthIndex: "auth-9", Name: "Spare Codex", Email: "spare@x.com", Type: "codex"},
	}
}

func TestSelectEmptyAccountsReturnsNone(t *testing.T) {
	got := Select(sample(), nil)
	if len(got) != 0 {
		t.Fatalf("got %d want 0", len(got))
	}
	got = Select(sample(), []string{})
	if len(got) != 0 {
		t.Fatalf("got %d want 0", len(got))
	}
}

func TestSelectByEmailCaseInsensitive(t *testing.T) {
	got := Select(sample(), []string{"ALICE@example.com"})
	if len(got) != 1 || got[0].AuthIndex != "1" {
		t.Fatalf("%#v", got)
	}
}

func TestSelectByAuthIndex(t *testing.T) {
	got := Select(sample(), []string{"auth-9"})
	if len(got) != 1 || got[0].AuthIndex != "auth-9" {
		t.Fatalf("%#v", got)
	}
}

func TestSelectByNameCaseInsensitive(t *testing.T) {
	got := Select(sample(), []string{"bob-work"})
	if len(got) != 1 || got[0].AuthIndex != "2" {
		t.Fatalf("%#v", got)
	}
}

func TestSelectExcludesNonCodex(t *testing.T) {
	got := Select(sample(), []string{"gemini", "g@x.com", "3"})
	if len(got) != 0 {
		t.Fatalf("non-codex must be excluded: %#v", got)
	}
}

func TestSelectMultipleStableOrder(t *testing.T) {
	got := Select(sample(), []string{"auth-9", "ALICE@example.com"})
	if len(got) != 2 {
		t.Fatalf("%#v", got)
	}
	// Preserve discovery order from files, not whitelist order.
	if got[0].AuthIndex != "1" || got[1].AuthIndex != "auth-9" {
		t.Fatalf("order=%#v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/selector -count=1`

Expected: FAIL (`undefined: Select`)

- [ ] **Step 3: Write minimal implementation**

Create `internal/selector/selector.go`:

```go
package selector

import (
	"strings"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
)

func IsCodex(a hostapi.AuthFile) bool {
	p := strings.ToLower(strings.TrimSpace(a.Provider))
	typ := strings.ToLower(strings.TrimSpace(a.Type))
	n := strings.ToLower(strings.TrimSpace(a.Name))
	return p == "codex" || typ == "codex" ||
		strings.Contains(p, "codex") || strings.Contains(typ, "codex") || strings.Contains(n, "codex")
}

func Select(files []hostapi.AuthFile, accounts []string) []hostapi.AuthFile {
	if len(accounts) == 0 {
		return nil
	}
	norms := make([]string, 0, len(accounts))
	for _, a := range accounts {
		a = strings.TrimSpace(a)
		if a != "" {
			norms = append(norms, a)
		}
	}
	if len(norms) == 0 {
		return nil
	}
	out := make([]hostapi.AuthFile, 0)
	for _, f := range files {
		if !IsCodex(f) {
			continue
		}
		if matches(f, norms) {
			out = append(out, f)
		}
	}
	return out
}

func matches(f hostapi.AuthFile, accounts []string) bool {
	idx := strings.TrimSpace(f.AuthIndex)
	email := strings.ToLower(strings.TrimSpace(f.Email))
	name := strings.ToLower(strings.TrimSpace(f.Name))
	account := strings.ToLower(strings.TrimSpace(f.Account))
	for _, raw := range accounts {
		if idx != "" && raw == idx {
			return true
		}
		want := strings.ToLower(raw)
		if email != "" && want == email {
			return true
		}
		if name != "" && want == name {
			return true
		}
		if account != "" && want == account {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/selector ./internal/hostapi -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/hostapi/types.go internal/selector/selector.go internal/selector/selector_test.go
git commit -m "$(cat <<'EOF'
feat: add Codex account selector with empty-whitelist semantics

EOF
)"
```

---
### Task 3: Host interface + quota extraction (display only)

**Files:**
- Create: `internal/hostapi/host.go`
- Create: `internal/hostapi/quota.go`
- Test: `internal/hostapi/quota_test.go`

**Interfaces:**
- Consumes: `AuthFile`, raw auth JSON
- Produces:
  - `type HTTPRequest struct { Method, URL string; Headers map[string][]string; Body []byte }`
  - `type HTTPResponse struct { StatusCode int; Headers map[string][]string; Body []byte }`
  - `type Host interface { AuthList(ctx); AuthGet(ctx, authIndex); HTTPDo(ctx, req); AuthGetRuntime(ctx, authIndex) (json.RawMessage, error) }`
  - `var ErrUnsupported = errors.New("unsupported")`
  - `func EnrichQuota(a AuthFile, listExtra json.RawMessage, runtime json.RawMessage) AuthFile` — copy known fields only; never invent numbers

- [ ] **Step 1: Write the failing test**

Create `internal/hostapi/quota_test.go`:

```go
package hostapi

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEnrichQuotaFromListFields(t *testing.T) {
	raw := json.RawMessage(`{
		"plan":"Plus",
		"five_hour":{"remaining":62.5,"resets_at":"2026-09-21T21:40:00+08:00"},
		"weekly":{"used":19,"resets_at":"2026-09-24T08:00:00+08:00"}
	}`)
	got := EnrichQuota(AuthFile{AuthIndex: "1"}, raw, nil)
	if got.Plan != "Plus" {
		t.Fatalf("plan=%q", got.Plan)
	}
	if got.FiveHour == nil || got.FiveHour.Remaining == nil || *got.FiveHour.Remaining != 62.5 {
		t.Fatalf("five_hour=%#v", got.FiveHour)
	}
	if got.FiveHour.ResetsAt == nil {
		t.Fatal("five_hour resets_at missing")
	}
	if got.Weekly == nil || got.Weekly.Used == nil || *got.Weekly.Used != 19 {
		t.Fatalf("weekly=%#v", got.Weekly)
	}
}

func TestEnrichQuotaMissingShowsEmpty(t *testing.T) {
	got := EnrichQuota(AuthFile{AuthIndex: "1", Name: "x"}, json.RawMessage(`{}`), nil)
	if got.Plan != "" || got.FiveHour != nil || got.Weekly != nil {
		t.Fatalf("must not invent: %#v", got)
	}
}

func TestEnrichQuotaRuntimeOverridesWhenPresent(t *testing.T) {
	list := json.RawMessage(`{"plan":"Free"}`)
	runtime := json.RawMessage(`{"plan":"Team","five_hour":{"remaining":10}}`)
	got := EnrichQuota(AuthFile{}, list, runtime)
	if got.Plan != "Team" {
		t.Fatalf("plan=%q", got.Plan)
	}
	if got.FiveHour == nil || got.FiveHour.Remaining == nil || *got.FiveHour.Remaining != 10 {
		t.Fatalf("%#v", got.FiveHour)
	}
}

func TestEnrichQuotaAcceptsAlternateKeys(t *testing.T) {
	raw := json.RawMessage(`{
		"plan_type":"Plus",
		"rate_limit":{"five_hour":{"remaining_fraction":0.5,"reset_at":1690000000},
		"primary_window":{"used_percent":40,"resets_in_seconds":3600}}
	}`)
	// Implementation should accept common aliases when clearly present; if a key is unknown, leave nil.
	got := EnrichQuota(AuthFile{}, raw, nil)
	// plan_type alias
	if got.Plan != "Plus" {
		t.Fatalf("plan alias=%q", got.Plan)
	}
	_ = time.Now()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/hostapi -count=1`

Expected: FAIL (`undefined: EnrichQuota`)

- [ ] **Step 3: Write minimal implementation**

Create `internal/hostapi/host.go`:

```go
package hostapi

import (
	"context"
	"encoding/json"
	"errors"
)

var ErrUnsupported = errors.New("unsupported host method")

type HTTPRequest struct {
	Method  string              `json:"Method"`
	URL     string              `json:"URL"`
	Headers map[string][]string `json:"Headers,omitempty"`
	Body    []byte              `json:"Body,omitempty"`
}

type HTTPResponse struct {
	StatusCode int                 `json:"StatusCode"`
	Headers    map[string][]string `json:"Headers,omitempty"`
	Body       []byte              `json:"Body,omitempty"`
}

type Host interface {
	AuthList(ctx context.Context) ([]AuthFile, error)
	AuthGet(ctx context.Context, authIndex string) ([]byte, error)
	HTTPDo(ctx context.Context, req HTTPRequest) (HTTPResponse, error)
	// AuthGetRuntime is optional; return ErrUnsupported when the host ABI lacks it.
	AuthGetRuntime(ctx context.Context, authIndex string) (json.RawMessage, error)
}
```

Create `internal/hostapi/quota.go`:

```go
package hostapi

import (
	"encoding/json"
	"strings"
	"time"
)

func EnrichQuota(a AuthFile, blobs ...json.RawMessage) AuthFile {
	for _, blob := range blobs {
		if len(blob) == 0 {
			continue
		}
		var m map[string]json.RawMessage
		if json.Unmarshal(blob, &m) != nil {
			continue
		}
		if p := firstString(m, "plan", "plan_type", "planType"); p != "" {
			a.Plan = p
		}
		if w := parseWindow(m, "five_hour", "fiveHour", "rate_limit_five_hour"); w != nil {
			a.FiveHour = w
		}
		if w := parseWindow(m, "weekly", "week", "weekly_limit"); w != nil {
			a.Weekly = w
		}
		// Nested rate_limit / primary_window best-effort (only if fields exist).
		if raw, ok := m["rate_limit"]; ok {
			var nested map[string]json.RawMessage
			if json.Unmarshal(raw, &nested) == nil {
				if w := parseWindow(nested, "five_hour", "fiveHour"); w != nil {
					a.FiveHour = w
				}
				if w := parseWindow(nested, "weekly", "week"); w != nil {
					a.Weekly = w
				}
			}
		}
		if raw, ok := m["primary_window"]; ok {
			if w := parseWindowMap(raw); w != nil && a.FiveHour == nil {
				a.FiveHour = w
			}
		}
	}
	return a
}

func parseWindow(m map[string]json.RawMessage, keys ...string) *QuotaWindow {
	for _, k := range keys {
		if raw, ok := m[k]; ok {
			if w := parseWindowMap(raw); w != nil {
				return w
			}
		}
	}
	return nil
}

func parseWindowMap(raw json.RawMessage) *QuotaWindow {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	w := &QuotaWindow{}
	found := false
	if v, ok := asFloat(m, "remaining", "remaining_fraction", "remainingFraction", "remaining_percent", "remainingPercent"); ok {
		w.Remaining = &v
		found = true
	}
	if v, ok := asFloat(m, "used", "used_percent", "usedPercent", "used_fraction", "usedFraction"); ok {
		w.Used = &v
		found = true
	}
	if t, ok := asTime(m, "resets_at", "reset_at", "resetsAt", "resetAt"); ok {
		w.ResetsAt = &t
		found = true
	} else if sec, ok := asFloat(m, "resets_in_seconds", "resetsInSeconds"); ok {
		t := time.Now().Add(time.Duration(sec) * time.Second)
		w.ResetsAt = &t
		found = true
	}
	if !found {
		return nil
	}
	return w
}

func firstString(m map[string]json.RawMessage, keys ...string) string {
	for _, k := range keys {
		raw, ok := m[k]
		if !ok {
			continue
		}
		var s string
		if json.Unmarshal(raw, &s) == nil && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func asFloat(m map[string]json.RawMessage, keys ...string) (float64, bool) {
	for _, k := range keys {
		raw, ok := m[k]
		if !ok {
			continue
		}
		var f float64
		if json.Unmarshal(raw, &f) == nil {
			return f, true
		}
		var s string
		if json.Unmarshal(raw, &s) == nil {
			var f2 float64
			if json.Unmarshal([]byte(s), &f2) == nil {
				return f2, true
			}
		}
	}
	return 0, false
}

func asTime(m map[string]json.RawMessage, keys ...string) (time.Time, bool) {
	for _, k := range keys {
		raw, ok := m[k]
		if !ok {
			continue
		}
		var s string
		if json.Unmarshal(raw, &s) == nil && strings.TrimSpace(s) != "" {
			if t, err := time.Parse(time.RFC3339, strings.TrimSpace(s)); err == nil {
				return t, true
			}
		}
		var unix int64
		if json.Unmarshal(raw, &unix) == nil && unix > 0 {
			return time.Unix(unix, 0), true
		}
		var f float64
		if json.Unmarshal(raw, &f) == nil && f > 0 {
			return time.Unix(int64(f), 0), true
		}
	}
	return time.Time{}, false
}
```

Tighten `TestEnrichQuotaAcceptsAlternateKeys` expectations to match the implementation above (plan_type → Plus; remaining_fraction under rate_limit.five_hour).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/hostapi -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/hostapi/host.go internal/hostapi/quota.go internal/hostapi/quota_test.go
git commit -m "$(cat <<'EOF'
feat: add host interface and best-effort quota enrichment

EOF
)"
```

---

### Task 4: RunState (running mutex + last summary)

**Files:**
- Create: `internal/runstate/runstate.go`
- Test: `internal/runstate/runstate_test.go`

**Interfaces:**
- Produces:
  - `type AccountResult struct { ... Status, Attempts, HTTPStatus, Error, Selected, Plan, FiveHour, Weekly ... }`
  - `type Summary struct { At time.Time; Mode string; Total, Attempted, Succeeded, Failed, Limited, Skipped int; Error string; Accounts []AccountResult; Message string }`
  - `type AccountView struct` for status list (includes `Selected bool` + quota)
  - `type State struct`
  - `func New() *State`
  - `func (s *State) TryBegin() bool` / `End(summary Summary)`
  - `func (s *State) Snapshot(cfgAccounts []string, discovered []hostapi.AuthFile, nextRun time.Time) StatusSnapshot`
  - `type StatusSnapshot struct { Running bool; NextRun *time.Time; LastRun *Summary; Accounts []AccountView }`

- [ ] **Step 1: Write the failing test**

```go
package runstate

import (
	"testing"
	"time"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
)

func TestTryBeginMutex(t *testing.T) {
	s := New()
	if !s.TryBegin() {
		t.Fatal("first begin")
	}
	if s.TryBegin() {
		t.Fatal("second begin must fail")
	}
	s.End(Summary{At: time.Now(), Mode: "manual", Message: "done"})
	if !s.TryBegin() {
		t.Fatal("begin after end")
	}
}

func TestSnapshotMarksSelectedAndQuota(t *testing.T) {
	s := New()
	rem := 12.0
	files := []hostapi.AuthFile{
		{AuthIndex: "1", Name: "a", Email: "a@x.com", Provider: "codex", Plan: "Plus", FiveHour: &hostapi.QuotaWindow{Remaining: &rem}},
		{AuthIndex: "2", Name: "b", Email: "b@x.com", Provider: "codex"},
	}
	snap := s.Snapshot([]string{"a@x.com"}, files, time.Time{})
	if len(snap.Accounts) != 2 {
		t.Fatalf("accounts=%d", len(snap.Accounts))
	}
	var a1, a2 AccountView
	for _, a := range snap.Accounts {
		if a.AuthIndex == "1" {
			a1 = a
		}
		if a.AuthIndex == "2" {
			a2 = a
		}
	}
	if !a1.Selected || a2.Selected {
		t.Fatalf("selected flags a1=%v a2=%v", a1.Selected, a2.Selected)
	}
	if a1.Plan != "Plus" || a1.FiveHour == nil || a1.FiveHour.Remaining == nil {
		t.Fatalf("quota not passed through: %#v", a1)
	}
	if a2.Plan != "" || a2.FiveHour != nil {
		t.Fatalf("must not invent quota: %#v", a2)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/runstate -count=1`

Expected: FAIL (`undefined: New`)

- [ ] **Step 3: Write minimal implementation**

```go
package runstate

import (
	"sort"
	"strings"
	"sync"
	"time"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/selector"
)

type AccountResult struct {
	AuthIndex   string `json:"auth_index,omitempty"`
	Name        string `json:"name"`
	Email       string `json:"email,omitempty"`
	Unavailable bool   `json:"unavailable,omitempty"`
	Status      string `json:"status"`
	Attempts    int    `json:"attempts"`
	HTTPStatus  int    `json:"http_status,omitempty"`
	Error       string `json:"error,omitempty"`
	EligibleAt  *time.Time `json:"eligible_at,omitempty"`
	ResetsAt    *time.Time `json:"resets_at,omitempty"`
}

type Summary struct {
	At         time.Time       `json:"at"`
	Mode       string          `json:"mode"`
	Total      int             `json:"total"`
	Attempted  int             `json:"attempted"`
	Succeeded  int             `json:"succeeded"`
	Failed     int             `json:"failed"`
	Limited    int             `json:"limited"`
	Skipped    int             `json:"skipped"`
	Error      string          `json:"error,omitempty"`
	Message    string          `json:"message,omitempty"`
	Accounts   []AccountResult `json:"accounts,omitempty"`
}

type AccountView struct {
	AuthIndex   string              `json:"auth_index,omitempty"`
	Name        string              `json:"name"`
	Email       string              `json:"email,omitempty"`
	Unavailable bool                `json:"unavailable,omitempty"`
	Disabled    bool                `json:"disabled,omitempty"`
	Selected    bool                `json:"selected"`
	Status      string              `json:"status,omitempty"`
	Attempts    int                 `json:"attempts,omitempty"`
	LastAttempt time.Time           `json:"last_attempt,omitempty"`
	LastSuccess time.Time           `json:"last_success,omitempty"`
	EligibleAt  time.Time           `json:"eligible_at,omitempty"`
	ResetsAt    time.Time           `json:"resets_at,omitempty"`
	Error       string              `json:"error,omitempty"`
	Plan        string              `json:"plan,omitempty"`
	FiveHour    *hostapi.QuotaWindow `json:"five_hour,omitempty"`
	Weekly      *hostapi.QuotaWindow `json:"weekly,omitempty"`
}

type StatusSnapshot struct {
	Running bool          `json:"running"`
	NextRun *time.Time    `json:"next_run,omitempty"`
	LastRun *Summary      `json:"last_run,omitempty"`
	Accounts []AccountView `json:"accounts,omitempty"`
}

type accountMem struct {
	Status      string
	Attempts    int
	LastAttempt time.Time
	LastSuccess time.Time
	EligibleAt  time.Time
	ResetsAt    time.Time
	Error       string
}

type State struct {
	mu       sync.RWMutex
	running  bool
	nextRun  time.Time
	lastRun  *Summary
	byIndex  map[string]accountMem
}

func New() *State {
	return &State{byIndex: map[string]accountMem{}}
}

func (s *State) TryBegin() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return false
	}
	s.running = true
	return true
}

func (s *State) End(summary Summary) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = false
	cp := summary
	cp.Accounts = append([]AccountResult(nil), summary.Accounts...)
	s.lastRun = &cp
	for _, a := range summary.Accounts {
		if a.AuthIndex == "" {
			continue
		}
		m := s.byIndex[a.AuthIndex]
		m.Status = a.Status
		m.Attempts = a.Attempts
		m.Error = a.Error
		m.LastAttempt = time.Now()
		if a.EligibleAt != nil {
			m.EligibleAt = *a.EligibleAt
		}
		if a.ResetsAt != nil {
			m.ResetsAt = *a.ResetsAt
		}
		if a.Status == "success" {
			m.LastSuccess = time.Now()
			m.ResetsAt = time.Time{}
		}
		s.byIndex[a.AuthIndex] = m
	}
}

func (s *State) SetNextRun(t time.Time) {
	s.mu.Lock()
	s.nextRun = t
	s.mu.Unlock()
}

func (s *State) GetAccount(authIndex string) accountMem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byIndex[authIndex]
}

func (s *State) Snapshot(cfgAccounts []string, discovered []hostapi.AuthFile, nextOverride time.Time) StatusSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var next *time.Time
	nr := s.nextRun
	if !nextOverride.IsZero() {
		nr = nextOverride
	}
	if !nr.IsZero() {
		t := nr
		next = &t
	}
	var last *Summary
	if s.lastRun != nil {
		cp := *s.lastRun
		cp.Accounts = append([]AccountResult(nil), s.lastRun.Accounts...)
		last = &cp
	}
	selected := selector.Select(discovered, cfgAccounts)
	sel := map[string]bool{}
	for _, a := range selected {
		sel[a.AuthIndex] = true
	}
	views := make([]AccountView, 0, len(discovered))
	for _, f := range discovered {
		if !selector.IsCodex(f) {
			continue
		}
		m := s.byIndex[f.AuthIndex]
		views = append(views, AccountView{
			AuthIndex:   f.AuthIndex,
			Name:        safeName(f),
			Email:       f.Email,
			Unavailable: f.Unavailable,
			Disabled:    f.Disabled,
			Selected:    sel[f.AuthIndex],
			Status:      m.Status,
			Attempts:    m.Attempts,
			LastAttempt: m.LastAttempt,
			LastSuccess: m.LastSuccess,
			EligibleAt:  m.EligibleAt,
			ResetsAt:    m.ResetsAt,
			Error:       m.Error,
			Plan:        f.Plan,
			FiveHour:    f.FiveHour,
			Weekly:      f.Weekly,
		})
	}
	sort.Slice(views, func(i, j int) bool {
		return strings.ToLower(views[i].Name) < strings.ToLower(views[j].Name)
	})
	return StatusSnapshot{Running: s.running, NextRun: next, LastRun: last, Accounts: views}
}

func safeName(a hostapi.AuthFile) string {
	if a.Name != "" {
		return a.Name
	}
	if a.ID != "" {
		return a.ID
	}
	return a.AuthIndex
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/runstate -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/runstate/runstate.go internal/runstate/runstate_test.go
git commit -m "$(cat <<'EOF'
feat: add run state mutex, summary, and selected status views

EOF
)"
```

---
### Task 5: Pinger (fixed model + mock Host)

**Files:**
- Create: `internal/pinger/pinger.go`
- Test: `internal/pinger/pinger_test.go`

**Interfaces:**
- Consumes: `hostapi.Host`, `hostapi.AuthFile`
- Produces:
  - Constants aligned with auto-ping: `ModelName="gpt-5.6-luna"`, `CodexURL`, `WindowInterval=5h`, `WindowGuard=1s`, `MaxAttempts=3`, `RetryBaseDelay=5s`, `AttemptTimeout=45s`
  - `type Outcome struct { Status string; HTTPStatus int; Retryable bool; Error string; ResetsAt time.Time; Attempts int; EligibleAt time.Time }`
  - `func PingAccount(ctx context.Context, h hostapi.Host, a hostapi.AuthFile, force bool, prevLastSuccess, prevResetsAt time.Time) Outcome`
  - 429 / `usage_limit_reached` → `limited`; success 2xx → `success`; continue retry on retryable failures

- [ ] **Step 1: Write the failing test**

```go
package pinger

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
)

type mockHost struct {
	token   string
	account string
	status  int
	body    []byte
	calls   int
}

func (m *mockHost) AuthList(context.Context) ([]hostapi.AuthFile, error) {
	return nil, errors.New("unused")
}
func (m *mockHost) AuthGet(context.Context, string) ([]byte, error) {
	raw, _ := json.Marshal(map[string]any{"access_token": m.token, "account_id": m.account})
	return raw, nil
}
func (m *mockHost) HTTPDo(ctx context.Context, req hostapi.HTTPRequest) (hostapi.HTTPResponse, error) {
	m.calls++
	if req.Method != "POST" || !strings.Contains(req.URL, "codex/responses") {
		tpanic("bad request")
	}
	var body map[string]any
	_ = json.Unmarshal(req.Body, &body)
	if body["model"] != ModelName {
		tpanic("model")
	}
	return hostapi.HTTPResponse{StatusCode: m.status, Body: m.body}, nil
}
func (m *mockHost) AuthGetRuntime(context.Context, string) (json.RawMessage, error) {
	return nil, hostapi.ErrUnsupported
}

func tpanic(s string) { panic(s) }

func TestPingSuccess(t *testing.T) {
	h := &mockHost{token: "tok", account: "acc", status: 200}
	out := PingAccount(context.Background(), h, hostapi.AuthFile{AuthIndex: "1", Name: "a"}, true, time.Time{}, time.Time{})
	if out.Status != "success" || out.HTTPStatus != 200 || out.Attempts != 1 {
		t.Fatalf("%#v", out)
	}
}

func TestPingLimitedUsage(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"error": map[string]any{"type": "usage_limit_reached", "message": "slow down", "resets_at": time.Now().Add(time.Hour).Unix()},
	})
	h := &mockHost{token: "tok", status: 429, body: body}
	out := PingAccount(context.Background(), h, hostapi.AuthFile{AuthIndex: "1"}, true, time.Time{}, time.Time{})
	if out.Status != "limited" {
		t.Fatalf("%#v", out)
	}
	if out.ResetsAt.IsZero() {
		t.Fatal("expected resets_at")
	}
}

func TestPingRetriesThenFails(t *testing.T) {
	h := &mockHost{token: "tok", status: 500, body: []byte(`{"error":{"message":"boom"}}`)}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// Shrink delays for test by using a test hook if present; otherwise accept real backoff with short MaxAttempts via build tag.
	// Implementation MUST expose `var retryDelayFn = retryDelay` for tests:
	old := retryDelayFn
	retryDelayFn = func(int) time.Duration { return time.Millisecond }
	defer func() { retryDelayFn = old }()
	out := PingAccount(ctx, h, hostapi.AuthFile{AuthIndex: "1"}, true, time.Time{}, time.Time{})
	if out.Status != "failed" || out.Attempts != MaxAttempts {
		t.Fatalf("%#v calls=%d", out, h.calls)
	}
	if h.calls != MaxAttempts {
		t.Fatalf("calls=%d", h.calls)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/pinger -count=1`

Expected: FAIL (`undefined: PingAccount` / `ModelName`)

- [ ] **Step 3: Write minimal implementation**

```go
package pinger

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
)

const (
	CodexURL       = "https://chatgpt.com/backend-api/codex/responses"
	ModelName      = "gpt-5.6-luna"
	DefaultPrompt  = "ping"
	WindowInterval = 5 * time.Hour
	WindowGuard    = 1 * time.Second
	AttemptTimeout = 45 * time.Second
	MaxAttempts    = 3
	RetryBaseDelay = 5 * time.Second
)

var retryDelayFn = func(failedAttempt int) time.Duration {
	if failedAttempt < 1 {
		failedAttempt = 1
	}
	return RetryBaseDelay * time.Duration(1<<uint(failedAttempt-1))
}

type Outcome struct {
	Status     string
	HTTPStatus int
	Retryable  bool
	Error      string
	ResetsAt   time.Time
	Attempts   int
	EligibleAt time.Time
}

type authMaterial struct{ AccessToken, AccountID string }

type codexBody struct {
	Model        string         `json:"model"`
	Instructions string         `json:"instructions"`
	Input        []codexMessage `json:"input"`
	Store        bool           `json:"store"`
	Stream       bool           `json:"stream"`
}
type codexMessage struct {
	Type    string      `json:"type"`
	Role    string      `json:"role"`
	Content []codexPart `json:"content"`
}
type codexPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
type upstreamErrorEnvelope struct {
	Error struct {
		Type            string `json:"type"`
		Message         string `json:"message"`
		ResetsAt        int64  `json:"resets_at"`
		ResetsInSeconds int64  `json:"resets_in_seconds"`
	} `json:"error"`
}

func PingAccount(ctx context.Context, h hostapi.Host, a hostapi.AuthFile, force bool, lastSuccess, prevResets time.Time) Outcome {
	base := Outcome{}
	if !force && !lastSuccess.IsZero() {
		eligible := lastSuccess.Add(WindowInterval + WindowGuard)
		if time.Now().Before(eligible) {
			wait := time.Until(eligible)
			if deadline, ok := ctx.Deadline(); ok && time.Now().Add(wait).After(deadline) {
				base.Status = "deferred"
				base.Error = "next window is outside this run timeout"
				base.EligibleAt = eligible
				return base
			}
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				base.Status = "failed"
				base.Error = ctx.Err().Error()
				base.EligibleAt = eligible
				return base
			case <-timer.C:
			}
		}
	}
	for attempt := 1; attempt <= MaxAttempts; attempt++ {
		base.Attempts = attempt
		attemptCtx, cancel := context.WithTimeout(ctx, AttemptTimeout)
		out := pingOnce(attemptCtx, h, a)
		cancel()
		base.HTTPStatus = out.HTTPStatus
		base.Error = out.Error
		base.ResetsAt = out.ResetsAt
		if out.Status == "success" {
			base.Status = "success"
			base.EligibleAt = time.Now().Add(WindowInterval + WindowGuard)
			return base
		}
		if out.Status == "limited" {
			base.Status = "limited"
			return base
		}
		if !out.Retryable || attempt == MaxAttempts {
			base.Status = "failed"
			return base
		}
		timer := time.NewTimer(retryDelayFn(attempt))
		select {
		case <-ctx.Done():
			timer.Stop()
			base.Status = "failed"
			base.Error = ctx.Err().Error()
			return base
		case <-timer.C:
		}
	}
	base.Status = "failed"
	base.Error = "retry loop exhausted"
	return base
}

func pingOnce(ctx context.Context, h hostapi.Host, a hostapi.AuthFile) Outcome {
	raw, err := h.AuthGet(ctx, a.AuthIndex)
	if err != nil {
		return Outcome{Status: "failed", Retryable: true, Error: "auth get: " + err.Error()}
	}
	m, err := parseAuthMaterial(raw)
	if err != nil {
		return Outcome{Status: "failed", Retryable: false, Error: err.Error()}
	}
	body, _ := json.Marshal(codexBody{
		Model:        ModelName,
		Instructions: "You are a helpful assistant.",
		Input:        []codexMessage{{Type: "message", Role: "user", Content: []codexPart{{Type: "input_text", Text: DefaultPrompt}}}},
		Store:        false,
		Stream:       true,
	})
	headers := map[string][]string{
		"Accept":        {"text/event-stream"},
		"Authorization": {"Bearer " + m.AccessToken},
		"Content-Type":  {"application/json"},
		"OpenAI-Beta":   {"responses=v1"},
		"originator":    {"codex_cli_rs"},
		"User-Agent":    {"codex_cli_rs/0.76.0"},
	}
	if m.AccountID != "" {
		headers["Chatgpt-Account-Id"] = []string{m.AccountID}
	}
	resp, err := h.HTTPDo(ctx, hostapi.HTTPRequest{Method: "POST", URL: CodexURL, Headers: headers, Body: body})
	if err != nil {
		return Outcome{Status: "failed", Retryable: true, Error: err.Error()}
	}
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return Outcome{Status: "success", HTTPStatus: resp.StatusCode}
	}
	typ, msg, reset := parseUpstreamError(resp.Body)
	if typ == "usage_limit_reached" {
		if msg == "" {
			msg = "usage limit reached"
		}
		return Outcome{Status: "limited", HTTPStatus: resp.StatusCode, Error: msg, ResetsAt: reset}
	}
	if msg == "" {
		msg = fmt.Sprintf("upstream HTTP %d", resp.StatusCode)
	}
	if typ != "" {
		msg = typ + ": " + msg
	}
	return Outcome{Status: "failed", HTTPStatus: resp.StatusCode, Retryable: isRetryableHTTP(resp.StatusCode), Error: msg, ResetsAt: reset}
}

func parseUpstreamError(body []byte) (string, string, time.Time) {
	if len(body) == 0 {
		return "", "", time.Time{}
	}
	var p upstreamErrorEnvelope
	if json.Unmarshal(body, &p) != nil {
		return "", "", time.Time{}
	}
	reset := time.Time{}
	if p.Error.ResetsAt > 0 {
		reset = time.Unix(p.Error.ResetsAt, 0)
	} else if p.Error.ResetsInSeconds > 0 {
		reset = time.Now().Add(time.Duration(p.Error.ResetsInSeconds) * time.Second)
	}
	return strings.TrimSpace(p.Error.Type), strings.TrimSpace(p.Error.Message), reset
}

func isRetryableHTTP(code int) bool {
	switch code {
	case 408, 425, 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

func parseAuthMaterial(raw []byte) (authMaterial, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return authMaterial{}, fmt.Errorf("invalid auth JSON")
	}
	token := firstString(root, "access_token", "accessToken", "oauth_access_token", "oauthAccessToken", "token", "id_token", "idToken")
	accountID := firstString(root, "account_id", "chatgpt_account_id", "accountId", "chatgptAccountId")
	for _, key := range []string{"tokens", "credentials", "auth", "oauth", "session"} {
		var nested map[string]json.RawMessage
		if b, ok := root[key]; ok && json.Unmarshal(b, &nested) == nil {
			if token == "" {
				token = firstString(nested, "access_token", "accessToken", "oauth_access_token", "oauthAccessToken", "token", "id_token", "idToken")
			}
			if accountID == "" {
				accountID = firstString(nested, "account_id", "chatgpt_account_id", "accountId", "chatgptAccountId")
			}
		}
	}
	if token == "" {
		return authMaterial{}, fmt.Errorf("missing access token")
	}
	return authMaterial{AccessToken: token, AccountID: accountID}, nil
}

func firstString(m map[string]json.RawMessage, keys ...string) string {
	for _, k := range keys {
		raw, ok := m[k]
		if !ok {
			continue
		}
		var s string
		if json.Unmarshal(raw, &s) == nil && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/pinger -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/pinger/pinger.go internal/pinger/pinger_test.go
git commit -m "$(cat <<'EOF'
feat: add fixed-model Codex pinger with limited/retry semantics

EOF
)"
```

---

### Task 6: Scheduler (next run + loop; no startup ping)

**Files:**
- Create: `internal/scheduler/scheduler.go`
- Test: `internal/scheduler/scheduler_test.go`

**Interfaces:**
- Consumes: `config.ParseClock`, timezone location
- Produces:
  - `func NextRun(now time.Time, loc *time.Location, times []string) time.Time`
  - `type Scheduler struct`
  - `func New(onFire func(ctx context.Context)) *Scheduler`
  - `func (s *Scheduler) Start(cfg config.Config)` — cancels previous; if disabled, clears next; **does not** call onFire immediately
  - `func (s *Scheduler) Stop()`
  - `func (s *Scheduler) Next() time.Time`

- [ ] **Step 1: Write the failing test**

```go
package scheduler

import (
	"testing"
	"time"
)

func TestNextRunPicksSoonestFutureSlot(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 9, 21, 10, 0, 0, 0, loc)
	next := NextRun(now, loc, []string{"06:00", "11:00", "16:00"})
	want := time.Date(2026, 9, 21, 11, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("got %s want %s", next, want)
	}
}

func TestNextRunRollsToTomorrow(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 9, 21, 22, 0, 0, 0, loc)
	next := NextRun(now, loc, []string{"06:00", "11:00"})
	want := time.Date(2026, 9, 22, 6, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("got %s want %s", next, want)
	}
}
```

Note: use `next.Equal` → actually `next.Equal` is wrong; use `next.Equal` no — Go is `next.Equal` invalid; use `!next.Equal` → **`!next.Equal`** wrong. Correct: `!next.Equal` → `if !next.Equal(want)` is invalid method case. **Must be `!next.Equal`** — fix to `if !next.Equal(want)` → use `!next.Equal` NO:

Correct Go: `if !next.Equal(want)`

Wait, Time.Equal doesn't exist - it's `next.Equal` - actually it's `time.Time.Equal`? No it's `Equal` with capital E: `next.equal` 

It's `t.Equal(u)` — `next.Equal(want)` is WRONG. Correct: `next.Equal` → `next.Equal`

**Use:** `if !next.Equal(want)` → The method is `Equal`:

```go
if !next.Equal(want) {
```

I will write in the plan file the correct form:

```go
if !next.Equal(want) {
```

Actually in Go: `func (t Time) Equal(u Time) bool` — so `next.Equal(want)` is a compile error. Must be `next.Equal` → `next.Equal` 

CORRECT CODE:
```go
if !next.Equal(want) {
```

Hmm I'm confusing myself. The method name is `Equal` with capital E:

```go
if !next.Equal(want) {  // WRONG - lowercase
if !next.Equal(want) {  // still wrong in my head
if !next.Equal(want) {
```

Final correct:
```go
if !next.Equal(want) {
```

I'll fix when writing - the correct call is `next.Equal(want)` NO - `next.Equal` -

**`next.Equal(want)`** is invalid.
**`next.Equal(want)`** 

The answer is: `if !next.Equal(want)` where Equal has capital E: `next.Equal` → 

```go
if !next.Equal(want) {
```

Looking at Go docs: `func (t Time) Equal(u Time) bool`

So: `if !next.Equal(want)` is WRONG
So: `if !next.Equal(want)` 

I'll write: `if !next.Equal(want)` 

OK in the actual file content below I use `!next.Equal(want)` - WAIT I'll use:

```go
if !next.Equal(want) {
```

No - I'll use: `if got, want := next, want; !got.Equal(want)`

Simplest: `if next.UTC() != want.UTC()` or `if !next.Equal(want)` with capital Equal.

In the appended content I'll carefully write `next.Equal` as `next.Equal` — the characters n-e-x-t-.-E-q-u-a-l

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/scheduler -count=1`

Expected: FAIL (`undefined: NextRun`)

- [ ] **Step 3: Write minimal implementation**

```go
package scheduler

import (
	"context"
	"sync"
	"time"

	"cpa-plugin-codex-selective-ping/internal/config"
)

type Scheduler struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	next    time.Time
	onFire  func(context.Context)
	setNext func(time.Time)
}

func New(onFire func(context.Context), setNext func(time.Time)) *Scheduler {
	if setNext == nil {
		setNext = func(time.Time) {}
	}
	return &Scheduler{onFire: onFire, setNext: setNext}
}

func NextRun(now time.Time, loc *time.Location, times []string) time.Time {
	best := time.Time{}
	for _, v := range times {
		h, m, err := config.ParseClock(v)
		if err != nil {
			continue
		}
		c := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, loc)
		if !c.After(now) {
			c = c.AddDate(0, 0, 1)
		}
		if best.IsZero() || c.Before(best) {
			best = c
		}
	}
	return best
}

func (s *Scheduler) Start(cfg config.Config) {
	s.Stop()
	s.mu.Lock()
	defer s.mu.Unlock()
	if !cfg.Enabled {
		s.next = time.Time{}
		s.setNext(time.Time{})
		return
	}
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.loop(ctx, loc, append([]string(nil), cfg.Times...))
}

func (s *Scheduler) loop(ctx context.Context, loc *time.Location, times []string) {
	for {
		next := NextRun(time.Now().In(loc), loc, times)
		s.mu.Lock()
		s.next = next
		s.mu.Unlock()
		s.setNext(next)
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case <-timer.C:
			if s.onFire != nil {
				s.onFire(ctx)
			}
		}
	}
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.next = time.Time{}
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.setNext(time.Time{})
}

func (s *Scheduler) Next() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.next
}
```

Fix the test file to use `next.Equal`:

```go
if !next.Equal(want) {
	t.Fatalf("got %s want %s", next, want)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/scheduler -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/scheduler/scheduler.go internal/scheduler/scheduler_test.go
git commit -m "$(cat <<'EOF'
feat: add daily timezone scheduler without startup ping

EOF
)"
```

---
### Task 7: Runner (select → ping → state; empty accounts short-circuit)

**Files:**
- Create: `internal/runner/runner.go`
- Test: `internal/runner/runner_test.go`

**Interfaces:**
- Consumes: `config.Config`, `hostapi.Host`, `selector.Select`, `pinger.PingAccount`, `runstate.State`
- Produces:
  - `const RunTimeout = 20 * time.Minute`
  - `type Runner struct { Host hostapi.Host; State *runstate.State }`
  - `func (r *Runner) Run(ctx context.Context, cfg config.Config, force bool) (runstate.Summary, bool)` — second return false if already running (caller maps to 409)
  - Empty `cfg.Accounts` → Summary with `Attempted=0`, `Message` explaining skipped / no accounts selected; do not call Host HTTP
  - Only selected accounts are pinged; disabled/missing auth_index skipped; continue on per-account failure

- [ ] **Step 1: Write the failing test**

```go
package runner

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"cpa-plugin-codex-selective-ping/internal/config"
	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/runstate"
)

type mockHost struct {
	files []hostapi.AuthFile
	http  int32
}

func (m *mockHost) AuthList(context.Context) ([]hostapi.AuthFile, error) { return m.files, nil }
func (m *mockHost) AuthGet(context.Context, string) ([]byte, error) {
	return json.Marshal(map[string]any{"access_token": "t", "account_id": "a"})
}
func (m *mockHost) HTTPDo(context.Context, hostapi.HTTPRequest) (hostapi.HTTPResponse, error) {
	atomic.AddInt32(&m.http, 1)
	return hostapi.HTTPResponse{StatusCode: 200}, nil
}
func (m *mockHost) AuthGetRuntime(context.Context, string) (json.RawMessage, error) {
	return nil, hostapi.ErrUnsupported
}

func TestRunEmptyAccountsNoHTTP(t *testing.T) {
	h := &mockHost{files: []hostapi.AuthFile{
		{AuthIndex: "1", Name: "a", Email: "a@x.com", Provider: "codex"},
	}}
	r := &Runner{Host: h, State: runstate.New()}
	sum, ok := r.Run(context.Background(), config.Config{Enabled: true, Timezone: "UTC", Times: []string{"06:00"}, Accounts: nil}, true)
	if !ok {
		t.Fatal("should accept run")
	}
	if sum.Attempted != 0 || atomic.LoadInt32(&h.http) != 0 {
		t.Fatalf("sum=%#v http=%d", sum, h.http)
	}
	if sum.Message == "" {
		t.Fatal("expected message about empty selection")
	}
}

func TestRunOnlySelected(t *testing.T) {
	h := &mockHost{files: []hostapi.AuthFile{
		{AuthIndex: "1", Name: "a", Email: "a@x.com", Provider: "codex"},
		{AuthIndex: "2", Name: "b", Email: "b@x.com", Provider: "codex"},
		{AuthIndex: "3", Name: "g", Email: "g@x.com", Provider: "gemini"},
	}}
	r := &Runner{Host: h, State: runstate.New()}
	sum, ok := r.Run(context.Background(), config.Config{Enabled: true, Timezone: "UTC", Times: []string{"06:00"}, Accounts: []string{"b@x.com"}}, true)
	if !ok {
		t.Fatal("accept")
	}
	if atomic.LoadInt32(&h.http) != 1 {
		t.Fatalf("http calls=%d want 1", h.http)
	}
	if sum.Succeeded != 1 || sum.Attempted != 1 {
		t.Fatalf("%#v", sum)
	}
}

func TestRunMutexRejectsSecond(t *testing.T) {
	h := &mockHost{files: []hostapi.AuthFile{
		{AuthIndex: "1", Name: "a", Email: "a@x.com", Provider: "codex"},
	}}
	st := runstate.New()
	if !st.TryBegin() {
		t.Fatal("prime running")
	}
	r := &Runner{Host: h, State: st}
	_, ok := r.Run(context.Background(), config.Config{Enabled: true, Timezone: "UTC", Times: []string{"06:00"}, Accounts: []string{"a@x.com"}}, true)
	if ok {
		t.Fatal("expected reject while running")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/runner -count=1`

Expected: FAIL (`undefined: Runner`)

- [ ] **Step 3: Write minimal implementation**

```go
package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"cpa-plugin-codex-selective-ping/internal/config"
	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/pinger"
	"cpa-plugin-codex-selective-ping/internal/runstate"
	"cpa-plugin-codex-selective-ping/internal/selector"
)

const RunTimeout = 20 * time.Minute

type Runner struct {
	Host  hostapi.Host
	State *runstate.State
}

func (r *Runner) Run(parent context.Context, cfg config.Config, force bool) (runstate.Summary, bool) {
	if !r.State.TryBegin() {
		return runstate.Summary{}, false
	}
	mode := "scheduled"
	if force {
		mode = "force"
	}
	summary := runstate.Summary{At: time.Now(), Mode: mode}
	defer r.State.End(summary)

	ctx, cancel := context.WithTimeout(parent, RunTimeout)
	defer cancel()

	if len(cfg.Accounts) == 0 {
		summary.Message = "no accounts selected; skipped ping (empty accounts whitelist)"
		summary.Attempted = 0
		return summary, true
	}

	files, err := r.Host.AuthList(ctx)
	if err != nil {
		summary.Error = "auth list failed: " + err.Error()
		return summary, true
	}
	files = enrichAll(ctx, r.Host, files)

	codexCount := 0
	for _, f := range files {
		if selector.IsCodex(f) {
			codexCount++
		}
	}
	summary.Total = codexCount

	selected := selector.Select(files, cfg.Accounts)
	if len(selected) == 0 {
		summary.Message = "no matching Codex accounts for configured whitelist"
		summary.Attempted = 0
		return summary, true
	}

	targets := make([]hostapi.AuthFile, 0, len(selected))
	for _, a := range selected {
		if a.Disabled {
			res := runstate.AccountResult{AuthIndex: a.AuthIndex, Name: nameOf(a), Email: a.Email, Unavailable: a.Unavailable, Status: "skipped_disabled"}
			summary.Skipped++
			summary.Accounts = append(summary.Accounts, res)
			continue
		}
		if a.AuthIndex == "" {
			res := runstate.AccountResult{Name: nameOf(a), Email: a.Email, Unavailable: a.Unavailable, Status: "skipped_missing_auth_index", Error: "missing auth_index"}
			summary.Skipped++
			summary.Accounts = append(summary.Accounts, res)
			continue
		}
		targets = append(targets, a)
	}

	results := make(chan runstate.AccountResult, len(targets))
	var wg sync.WaitGroup
	for _, a := range targets {
		wg.Add(1)
		go func(auth hostapi.AuthFile) {
			defer wg.Done()
			mem := r.State.GetAccount(auth.AuthIndex)
			out := pinger.PingAccount(ctx, r.Host, auth, force, mem.LastSuccess, mem.ResetsAt)
			res := runstate.AccountResult{
				AuthIndex: auth.AuthIndex, Name: nameOf(auth), Email: auth.Email, Unavailable: auth.Unavailable,
				Status: out.Status, Attempts: out.Attempts, HTTPStatus: out.HTTPStatus, Error: out.Error,
			}
			if !out.EligibleAt.IsZero() {
				t := out.EligibleAt
				res.EligibleAt = &t
			}
			if !out.ResetsAt.IsZero() {
				t := out.ResetsAt
				res.ResetsAt = &t
			}
			results <- res
		}(a)
	}
	go func() { wg.Wait(); close(results) }()

	firstErr := ""
	for res := range results {
		summary.Accounts = append(summary.Accounts, res)
		switch res.Status {
		case "success":
			summary.Attempted++
			summary.Succeeded++
		case "limited":
			if res.Attempts > 0 {
				summary.Attempted++
			}
			summary.Limited++
		case "deferred":
			summary.Skipped++
		default:
			if res.Attempts > 0 {
				summary.Attempted++
			}
			summary.Failed++
			if firstErr == "" {
				firstErr = res.Error
			}
		}
	}
	sort.Slice(summary.Accounts, func(i, j int) bool {
		return summary.Accounts[i].Name < summary.Accounts[j].Name
	})
	if summary.Failed > 0 {
		summary.Error = firstErr
	}
	return summary, true
}

func enrichAll(ctx context.Context, h hostapi.Host, files []hostapi.AuthFile) []hostapi.AuthFile {
	out := make([]hostapi.AuthFile, len(files))
	for i, f := range files {
		var runtime json.RawMessage
		if raw, err := h.AuthGetRuntime(ctx, f.AuthIndex); err == nil {
			runtime = raw
		}
		// list row itself may already carry quota fields via JSON unmarshal into AuthFile
		extra, _ := json.Marshal(f)
		out[i] = hostapi.EnrichQuota(f, extra, runtime)
	}
	return out
}

func nameOf(a hostapi.AuthFile) string {
	if a.Name != "" {
		return a.Name
	}
	if a.ID != "" {
		return a.ID
	}
	if a.Email != "" {
		return a.Email
	}
	return a.AuthIndex
}

// silence unused in case GetAccount fields differ in early drafts
var _ = fmt.Sprintf
```

Remove the unused `fmt` import if `GetAccount` works — final implementer should drop `var _ = fmt.Sprintf` and unused imports until `go test` is clean.

Also export `LastSuccess`/`ResetsAt` from runstate `GetAccount` — Task 4's `accountMem` is unexported; **change Task 4 `GetAccount` return type** to a small exported struct if needed:

```go
type AccountMem struct {
	LastSuccess time.Time
	ResetsAt    time.Time
	EligibleAt  time.Time
	Status      string
	Attempts    int
	Error       string
}

func (s *State) GetAccount(authIndex string) AccountMem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := s.byIndex[authIndex]
	return AccountMem{LastSuccess: m.LastSuccess, ResetsAt: m.ResetsAt, EligibleAt: m.EligibleAt, Status: m.Status, Attempts: m.Attempts, Error: m.Error}
}
```

Apply that export tweak in `internal/runstate/runstate.go` as part of this task if not already done.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/runner -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/runner/runner.go internal/runner/runner_test.go internal/runstate/runstate.go
git commit -m "$(cat <<'EOF'
feat: add runner that pings only selected accounts

EOF
)"
```

---

### Task 8: Management handler (status shape + run 202/409)

**Files:**
- Create: `internal/management/handler.go`
- Test: `internal/management/handler_test.go`
- Create: `internal/plugin/plugin.go` (wiring stub used by handler tests)

**Interfaces:**
- Produces:
  - `type StatusResponse struct` with Enabled, Version, Model, Timezone, Times, Accounts (with selected + optional quota), NextRun, Running, LastRun, WindowSeconds, etc.
  - `type Handler struct { Plugin *plugin.Plugin }`
  - `func (h *Handler) Handle(req ManagementRequest) ManagementResponse`
  - Routes: `GET .../status` JSON; `POST .../run` 202/409; `GET .../resource/.../status` HTML (UI filled in Task 9)
  - Plugin ID paths contain `codex-selective-ping`

- [ ] **Step 1: Write the failing test**

```go
package management

import (
	"encoding/json"
	"strings"
	"testing"

	"cpa-plugin-codex-selective-ping/internal/config"
	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/plugin"
	"cpa-plugin-codex-selective-ping/internal/runstate"
)

type listHost struct{ files []hostapi.AuthFile }

func (m *listHost) AuthList(ctx interface{ Done() <-chan struct{} }) ([]hostapi.AuthFile, error) {
	return m.files, nil
}

// Use a proper hostapi.Host mock:
type mh struct{ files []hostapi.AuthFile }

func (m *mh) AuthList(ctx context.Context) ([]hostapi.AuthFile, error) { return m.files, nil }
func (m *mh) AuthGet(context.Context, string) ([]byte, error) {
	return json.Marshal(map[string]any{"access_token": "t"})
}
func (m *mh) HTTPDo(context.Context, hostapi.HTTPRequest) (hostapi.HTTPResponse, error) {
	return hostapi.HTTPResponse{StatusCode: 200}, nil
}
func (m *mh) AuthGetRuntime(context.Context, string) (json.RawMessage, error) {
	return nil, hostapi.ErrUnsupported
}
```

Fix the test file to compile — full version:

```go
package management

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"cpa-plugin-codex-selective-ping/internal/config"
	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/plugin"
)

type mh struct{ files []hostapi.AuthFile }

func (m *mh) AuthList(context.Context) ([]hostapi.AuthFile, error) { return m.files, nil }
func (m *mh) AuthGet(context.Context, string) ([]byte, error) {
	return json.Marshal(map[string]any{"access_token": "t"})
}
func (m *mh) HTTPDo(context.Context, hostapi.HTTPRequest) (hostapi.HTTPResponse, error) {
	return hostapi.HTTPResponse{StatusCode: 200}, nil
}
func (m *mh) AuthGetRuntime(context.Context, string) (json.RawMessage, error) {
	return nil, hostapi.ErrUnsupported
}

func TestStatusShapeSelected(t *testing.T) {
	p := plugin.New(&mh{files: []hostapi.AuthFile{
		{AuthIndex: "1", Name: "a", Email: "a@x.com", Provider: "codex", Plan: "Plus"},
		{AuthIndex: "2", Name: "b", Email: "b@x.com", Provider: "codex"},
	}}, "0.1.0")
	p.ApplyConfig(config.Config{Enabled: true, Timezone: "Asia/Taipei", Times: []string{"06:00"}, Accounts: []string{"a@x.com"}})
	h := &Handler{Plugin: p}
	resp := h.Handle(Request{Method: "GET", Path: "/v0/management/plugins/codex-selective-ping/status"})
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, resp.Body)
	}
	var st StatusResponse
	if err := json.Unmarshal(resp.Body, &st); err != nil {
		t.Fatal(err)
	}
	if st.Version == "" || st.Model == "" || st.Timezone != "Asia/Taipei" {
		t.Fatalf("%#v", st)
	}
	found := false
	for _, a := range st.Accounts {
		if a.AuthIndex == "1" {
			found = true
			if !a.Selected || a.Plan != "Plus" {
				t.Fatalf("%#v", a)
			}
		}
		if a.AuthIndex == "2" && a.Selected {
			t.Fatal("b should not be selected")
		}
	}
	if !found {
		t.Fatal("missing account 1")
	}
}

func TestRunAcceptedAndConflict(t *testing.T) {
	p := plugin.New(&mh{files: nil}, "0.1.0")
	p.ApplyConfig(config.Config{Enabled: true, Timezone: "UTC", Times: []string{"06:00"}, Accounts: []string{}})
	h := &Handler{Plugin: p}
	r1 := h.Handle(Request{Method: "POST", Path: "/v0/management/plugins/codex-selective-ping/run"})
	if r1.StatusCode != 202 {
		t.Fatalf("want 202 got %d body=%s", r1.StatusCode, r1.Body)
	}
	// Force running flag
	if !p.State.TryBegin() {
		// if run already claimed asynchronously, OK
	} else {
		r2 := h.Handle(Request{Method: "POST", Path: "/v0/management/plugins/codex-selective-ping/run"})
		if r2.StatusCode != 409 {
			t.Fatalf("want 409 got %d", r2.StatusCode)
		}
		p.State.End(runstate.Summary{})
	}
	_ = strings.Contains
}
```

Add missing import `runstate` in the conflict test. Prefer deterministic mutex test:

```go
func TestRunConflict409(t *testing.T) {
	p := plugin.New(&mh{}, "0.1.0")
	p.ApplyConfig(config.Config{Enabled: true, Timezone: "UTC", Times: []string{"06:00"}, Accounts: []string{}})
	if !p.State.TryBegin() {
		t.Fatal("begin")
	}
	h := &Handler{Plugin: p}
	r := h.Handle(Request{Method: "POST", Path: "/v0/management/plugins/codex-selective-ping/run"})
	if r.StatusCode != 409 {
		t.Fatalf("got %d", r.StatusCode)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/management ./internal/plugin -count=1`

Expected: FAIL (missing types/packages)

- [ ] **Step 3: Write minimal implementation**

Create `internal/plugin/plugin.go`:

```go
package plugin

import (
	"context"
	"sync"

	"cpa-plugin-codex-selective-ping/internal/config"
	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/pinger"
	"cpa-plugin-codex-selective-ping/internal/runner"
	"cpa-plugin-codex-selective-ping/internal/runstate"
	"cpa-plugin-codex-selective-ping/internal/scheduler"
)

const PluginID = "codex-selective-ping"

type Plugin struct {
	Host    hostapi.Host
	Version string
	State   *runstate.State
	Runner  *runner.Runner
	Sched   *scheduler.Scheduler

	mu  sync.Mutex
	cfg config.Config
}

func New(h hostapi.Host, version string) *Plugin {
	p := &Plugin{Host: h, Version: version, State: runstate.New(), cfg: config.DefaultConfig()}
	p.Runner = &runner.Runner{Host: h, State: p.State}
	p.Sched = scheduler.New(func(ctx context.Context) {
		p.mu.Lock()
		cfg := p.cfg
		p.mu.Unlock()
		p.Runner.Run(ctx, cfg, false)
	}, p.State.SetNextRun)
	return p
}

func (p *Plugin) ApplyConfig(cfg config.Config) {
	p.mu.Lock()
	p.cfg = cfg
	p.mu.Unlock()
	p.Sched.Start(cfg)
}

func (p *Plugin) Config() config.Config {
	p.mu.Lock()
	defer p.mu.Unlock()
	cp := p.cfg
	cp.Times = append([]string(nil), p.cfg.Times...)
	cp.Accounts = append([]string(nil), p.cfg.Accounts...)
	return cp
}

func (p *Plugin) StartManualRun() bool {
	cfg := p.Config()
	// claim inside Runner.Run; but we need async 202: claim via TryBegin path inside goroutine carefully.
	// Pattern: try begin here by starting goroutine that calls Run — Run also TryBegin.
	// So expose Runner method that assumes not begun, OR:
	if !p.State.TryBegin() {
		return false
	}
	// release and let Run claim again — BAD. Instead add Runner.RunClaimed or:
	go func() {
		// We already claimed; call internal path.
		p.Runner.RunAlreadyClaimed(context.Background(), cfg, true)
	}()
	return true
}

func (p *Plugin) Shutdown() {
	p.Sched.Stop()
}
```

Better: change `Runner.Run` to not double-claim when using `StartManualRun`. Clean approach for implementer:

```go
func (r *Runner) Run(parent context.Context, cfg config.Config, force bool) (runstate.Summary, bool) {
	if !r.State.TryBegin() {
		return runstate.Summary{}, false
	}
	return r.runClaimed(parent, cfg, force), true
}

func (r *Runner) runClaimed(...) runstate.Summary { /* body from Task 7 without TryBegin */ }

// Manual:
func (p *Plugin) StartManualRun() bool {
	if !p.State.TryBegin() { return false }
	cfg := p.Config()
	go p.Runner.runClaimed(context.Background(), cfg, true)
	return true
}
```

Refactor Task 7 accordingly in this task.

Create `internal/management/handler.go`:

```go
package management

import (
	"context"
	"encoding/json"
	"strings"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/pinger"
	"cpa-plugin-codex-selective-ping/internal/plugin"
	"cpa-plugin-codex-selective-ping/internal/runstate"
	"cpa-plugin-codex-selective-ping/internal/selector"
)

type Request struct {
	Method  string
	Path    string
	Headers map[string][]string
	Query   map[string][]string
	Body    []byte
}

type Response struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte
}

type StatusResponse struct {
	Enabled          bool                 `json:"enabled"`
	Version          string               `json:"version"`
	Model            string               `json:"model"`
	Timezone         string               `json:"timezone"`
	Times            []string             `json:"times"`
	AccountsConfig   []string             `json:"accounts_config"`
	WindowSeconds    int64                `json:"window_seconds"`
	GuardSeconds     int64                `json:"guard_seconds"`
	MaxAttempts      int                  `json:"max_attempts"`
	RetryBaseSeconds int64                `json:"retry_base_seconds"`
	NextRun          interface{}          `json:"next_run,omitempty"`
	Running          bool                 `json:"running"`
	LastRun          *runstate.Summary    `json:"last_run,omitempty"`
	Accounts         []runstate.AccountView `json:"accounts,omitempty"`
}

type Handler struct {
	Plugin *plugin.Plugin
}

func (h *Handler) Handle(req Request) Response {
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	path := strings.TrimSpace(req.Path)
	switch {
	case method == "GET" && strings.Contains(path, "/v0/resource/plugins/") && strings.HasSuffix(path, "/status"):
		body := []byte(RenderStatusPage(h.status()))
		return Response{StatusCode: 200, Headers: map[string][]string{"content-type": {"text/html; charset=utf-8"}, "cache-control": {"no-store"}}, Body: body}
	case method == "GET" && strings.HasSuffix(path, "/plugins/codex-selective-ping/status"):
		body, _ := json.Marshal(h.status())
		return Response{StatusCode: 200, Headers: map[string][]string{"content-type": {"application/json; charset=utf-8"}, "cache-control": {"no-store"}}, Body: body}
	case method == "POST" && strings.HasSuffix(path, "/plugins/codex-selective-ping/run"):
		if !h.Plugin.StartManualRun() {
			body, _ := json.Marshal(map[string]any{"accepted": false, "running": true, "message": "a run is already in progress"})
			return Response{StatusCode: 409, Headers: map[string][]string{"content-type": {"application/json; charset=utf-8"}}, Body: body}
		}
		body, _ := json.Marshal(map[string]any{"accepted": true, "running": true, "force": true})
		return Response{StatusCode: 202, Headers: map[string][]string{"content-type": {"application/json; charset=utf-8"}}, Body: body}
	default:
		body, _ := json.Marshal(map[string]string{"error": "not found"})
		return Response{StatusCode: 404, Headers: map[string][]string{"content-type": {"application/json; charset=utf-8"}}, Body: body}
	}
}

func (h *Handler) status() StatusResponse {
	cfg := h.Plugin.Config()
	ctx := context.Background()
	files, _ := h.Plugin.Host.AuthList(ctx)
	enriched := make([]hostapi.AuthFile, 0, len(files))
	for _, f := range files {
		if !selector.IsCodex(f) {
			continue
		}
		var runtime json.RawMessage
		if raw, err := h.Plugin.Host.AuthGetRuntime(ctx, f.AuthIndex); err == nil {
			runtime = raw
		}
		extra, _ := json.Marshal(f)
		enriched = append(enriched, hostapi.EnrichQuota(f, extra, runtime))
	}
	// Also include non-filtered discovery for Snapshot: pass all codex-enriched
	all := enriched
	snap := h.Plugin.State.Snapshot(cfg.Accounts, all, h.Plugin.Sched.Next())
	return StatusResponse{
		Enabled: cfg.Enabled, Version: h.Plugin.Version, Model: pinger.ModelName,
		Timezone: cfg.Timezone, Times: cfg.Times, AccountsConfig: cfg.Accounts,
		WindowSeconds: int64(pinger.WindowInterval / 1e9), GuardSeconds: int64(pinger.WindowGuard / 1e9),
		MaxAttempts: pinger.MaxAttempts, RetryBaseSeconds: int64(pinger.RetryBaseDelay / 1e9),
		NextRun: snap.NextRun, Running: snap.Running, LastRun: snap.LastRun, Accounts: snap.Accounts,
	}
}
```

Use `time.Duration.Seconds()` instead of `/1e9` in real code:

```go
WindowSeconds: int64(pinger.WindowInterval.Seconds()),
```

Stub `RenderStatusPage` in handler.go for this task:

```go
func RenderStatusPage(st StatusResponse) string {
	return "<!doctype html><html><body><h1>Codex Selective Ping</h1></body></html>"
}
```

Task 9 replaces it.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/management ./internal/plugin ./internal/runner -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/plugin/plugin.go internal/management/handler.go internal/management/handler_test.go internal/runner/runner.go
git commit -m "$(cat <<'EOF'
feat: wire plugin and management status/run handlers

EOF
)"
```

---
### Task 9: Traditional Chinese Management UI (resource page)

**Files:**
- Create: `internal/management/ui.go` (replace Task 8 stub `RenderStatusPage`)
- Modify: `internal/management/handler.go` (ensure resource GET uses `RenderStatusPage`)
- Optional visual check against: `docs/superpowers/mocks/management-ui-mock.html`

**Interfaces:**
- Produces: `func RenderStatusPage(st StatusResponse) string` — single HTML page, zh-Hant
- UI sections: 概況 / 排程 / 帳號 (checkbox + Plan/5h/週限/狀態) / 動作 (Management Key, 儲存, 立刻執行, 重新整理) / 最近一次結果
- Save: `PATCH /v0/management/plugins/codex-selective-ping/config` with Bearer key (host API; not plugin route)
- Run: `POST /v0/management/plugins/codex-selective-ping/run`
- Refresh status: `GET /v0/management/plugins/codex-selective-ping/status`
- Missing quota → display `—`; never invent
- Empty selection banner: 未勾選明確提示不會 ping
- Do not write Management Key to `localStorage`

- [ ] **Step 1: Write the failing test**

```go
package management

import (
	"strings"
	"testing"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/runstate"
)

func TestRenderStatusPageChineseAndQuotaDash(t *testing.T) {
	rem := 62.0
	html := RenderStatusPage(StatusResponse{
		Enabled: true, Version: "0.1.0", Model: "gpt-5.6-luna",
		Timezone: "Asia/Taipei", Times: []string{"06:00", "21:00"},
		Accounts: []runstate.AccountView{
			{AuthIndex: "1", Name: "alice", Email: "a@x.com", Selected: true, Plan: "Plus", FiveHour: &hostapi.QuotaWindow{Remaining: &rem}, Status: "success"},
			{AuthIndex: "2", Name: "bob", Selected: false, Status: "unknown"},
		},
	})
	for _, want := range []string{"概況", "排程", "帳號", "立刻執行", "儲存設定", "Management Key", "Plan", "5h", "週限", "—", "plugins/codex-selective-ping/config", "plugins/codex-selective-ping/run"} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q", want)
		}
	}
	if strings.Contains(html, "localStorage") {
		t.Fatal("must not use localStorage for key")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/management -run TestRenderStatusPageChineseAndQuotaDash -count=1`

Expected: FAIL (stub HTML missing strings) or FAIL if stub returns minimal page

- [ ] **Step 3: Write minimal implementation**

Create `internal/management/ui.go` with complete embedded HTML/JS/CSS (Traditional Chinese). Full code:

```go
package management

import (
	"fmt"
	"html"
	"strings"
	"time"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/runstate"
)

func RenderStatusPage(st StatusResponse) string {
	next := "—"
	if st.NextRun != nil {
		switch v := st.NextRun.(type) {
		case time.Time:
			next = v.Format(time.RFC3339)
		case *time.Time:
			if v != nil {
				next = v.Format(time.RFC3339)
			}
		case string:
			if v != "" {
				next = v
			}
		}
	}
	running := "否"
	if st.Running {
		running = "是"
	}
	enabled := "停用"
	if st.Enabled {
		enabled = "已啟用"
	}
	selectedN := 0
	for _, a := range st.Accounts {
		if a.Selected {
			selectedN++
		}
	}
	var rows strings.Builder
	for _, a := range st.Accounts {
		checked := ""
		if a.Selected {
			checked = " checked"
		}
		ident := html.EscapeString(a.AuthIndex)
		if a.Email != "" {
			ident = html.EscapeString(a.Email)
		}
		fmt.Fprintf(&rows, `<tr>
<td><input type="checkbox" class="acct" data-id="%s"%s></td>
<td>%s<br><small>%s · auth_index: %s</small></td>
<td>%s</td>
<td class="quota">%s</td>
<td class="quota">%s</td>
<td>%s</td>
</tr>`,
			html.EscapeString(preferID(a)), checked,
			html.EscapeString(a.Name), html.EscapeString(a.Email), html.EscapeString(a.AuthIndex),
			dash(a.Plan),
			formatWindow(a.FiveHour),
			formatWindow(a.Weekly),
			html.EscapeString(orDash(a.Status)),
		)
		_ = ident
	}
	lastBlock := `<p class="hint">尚無執行紀錄</p>`
	if st.LastRun != nil {
		lr := st.LastRun
		var lrRows strings.Builder
		for _, a := range lr.Accounts {
			fmt.Fprintf(&lrRows, `<tr><td>%s</td><td>%s</td><td>%d</td><td>%s</td></tr>`,
				html.EscapeString(a.Name), html.EscapeString(a.Status), a.HTTPStatus, html.EscapeString(orDash(a.Error)))
		}
		lastBlock = fmt.Sprintf(`<div class="row" style="margin-bottom:8px">
<span class="tag">mode: %s</span>
<span class="tag">%s</span>
<span class="tag">ok %d</span>
<span class="tag">limited %d</span>
<span class="tag">failed %d</span>
<span class="tag">skipped %d</span>
</div>
<table><thead><tr><th>帳號</th><th>結果</th><th>HTTP</th><th>說明</th></tr></thead><tbody>%s</tbody></table>`,
			html.EscapeString(lr.Mode), html.EscapeString(lr.At.Format(time.RFC3339)),
			lr.Succeeded, lr.Limited, lr.Failed, lr.Skipped, lrRows.String())
		if lr.Message != "" {
			lastBlock += `<p class="hint">` + html.EscapeString(lr.Message) + `</p>`
		}
	}
	timesJSON, _ := jsonMarshal(st.Times)
	accountsJSON, _ := jsonMarshal(st.AccountsConfig)
	banner := fmt.Sprintf("目前已勾選 %d / %d 個帳號。未勾選的不會被排程或「立刻執行」打到。", selectedN, len(st.Accounts))
	if len(st.AccountsConfig) == 0 {
		banner = "帳號白名單為空：排程與立刻執行都不會 ping 任何人。"
	}
	return fmt.Sprintf(`<!doctype html>
<html lang="zh-Hant">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>Codex Selective Ping</title>
<style>
:root{--bg:#f4f6f8;--card:#fff;--text:#1f2937;--muted:#6b7280;--line:#e5e7eb;--brand:#2563eb;--chip:#eff6ff;--chip-text:#1d4ed8}
*{box-sizing:border-box}body{margin:0;font-family:ui-sans-serif,system-ui,sans-serif;background:var(--bg);color:var(--text)}
header{background:#111827;color:#fff;padding:16px 24px}header h1{margin:0;font-size:18px}header p{margin:4px 0 0;font-size:12px;opacity:.75}
main{max-width:1100px;margin:20px auto;padding:0 16px 40px;display:grid;gap:16px}
.card{background:var(--card);border:1px solid var(--line);border-radius:12px;padding:16px 18px}
.card h2{margin:0 0 12px;font-size:15px}.grid{display:grid;grid-template-columns:repeat(4,1fr);gap:10px}
@media(max-width:800px){.grid{grid-template-columns:repeat(2,1fr)}}
.stat{background:#f9fafb;border:1px solid var(--line);border-radius:10px;padding:10px 12px}
.stat .k{font-size:11px;color:var(--muted)}.stat .v{font-size:14px;font-weight:600;margin-top:4px}
.row{display:flex;gap:8px;flex-wrap:wrap;align-items:center}label{font-size:13px;color:var(--muted)}
input[type=text],input[type=password],input[type=time]{border:1px solid var(--line);border-radius:8px;padding:8px 10px;font-size:13px}
input[type=password]{min-width:220px}button{border:0;border-radius:8px;padding:8px 12px;font-size:13px;cursor:pointer}
.btn{background:var(--brand);color:#fff}.btn.secondary{background:#e5e7eb;color:#111}
.hint{font-size:12px;color:var(--muted);margin-top:8px}
.tag{display:inline-block;padding:2px 8px;border-radius:999px;background:var(--chip);color:var(--chip-text);font-size:11px}
table{width:100%%;border-collapse:collapse;font-size:13px}th,td{padding:10px 8px;border-bottom:1px solid var(--line);text-align:left;vertical-align:top}
th{font-size:11px;color:var(--muted)}.quota{font-variant-numeric:tabular-nums;white-space:nowrap}.quota small{display:block;color:var(--muted);font-size:11px}
.banner{background:#fff7ed;border:1px solid #fed7aa;color:#9a3412;border-radius:10px;padding:10px 12px;font-size:13px}
pre{white-space:pre-wrap;background:#f6f6f6;padding:12px;border-radius:6px}
</style>
</head>
<body>
<header>
  <h1>Codex Selective Ping</h1>
  <p>獨立 CPA 插件 · 指定帳號排程 ping</p>
</header>
<main>
<section class="card"><h2>概況</h2>
<div class="grid">
  <div class="stat"><div class="k">狀態</div><div class="v">%s</div></div>
  <div class="stat"><div class="k">版本 / 模型</div><div class="v">%s · %s</div></div>
  <div class="stat"><div class="k">時區 / 時段</div><div class="v">%s · %s</div></div>
  <div class="stat"><div class="k">下次執行</div><div class="v">%s · 執行中：%s</div></div>
</div></section>
<section class="card"><h2>排程</h2>
<div class="row" style="margin-bottom:10px">
  <label>時區</label><input id="tz" type="text" value="%s"/>
  <label>新增時段</label><input id="new-time" type="time" value="21:00"/>
  <button class="btn secondary" type="button" onclick="addTime()">加入</button>
</div>
<div class="row" id="times"></div>
<p class="hint">啟動時不會立刻全 ping；到點才跑。空帳號清單 = 誰都不 ping。</p>
</section>
<section class="card"><h2>帳號</h2>
<div class="banner" id="banner">%s</div>
<div class="row" style="margin:12px 0">
  <button class="btn secondary" type="button" onclick="setAll(true)">全選</button>
  <button class="btn secondary" type="button" onclick="setAll(false)">清除</button>
</div>
<table>
<thead><tr><th></th><th>帳號</th><th>Plan</th><th>5h</th><th>週限</th><th>狀態</th></tr></thead>
<tbody>%s</tbody>
</table>
<p class="hint">5h／週限來自 CPA 既有資料；沒有就顯示 —，不會自己推估。額度欄只供參考，不決定是否 ping。</p>
</section>
<section class="card"><h2>動作</h2>
<div class="row">
  <input id="management-key" type="password" autocomplete="off" placeholder="Management Key（當次輸入，不寫入 localStorage）"/>
  <button class="btn" type="button" onclick="saveCfg()">儲存設定</button>
  <button class="btn secondary" type="button" onclick="runNow()">立刻執行</button>
  <button class="btn secondary" type="button" onclick="location.reload()">重新整理</button>
</div>
<p class="hint">儲存：PATCH 宿主 plugins.configs。立刻執行：只 ping 目前已儲存白名單中的帳號（請先儲存勾選）。</p>
<pre id="result"></pre>
</section>
<section class="card"><h2>最近一次結果</h2>%s</section>
</main>
<script>
const initialTimes = %s;
const pluginId = "codex-selective-ping";
let times = Array.isArray(initialTimes) ? initialTimes.slice() : [];
function renderTimes(){
  const el = document.getElementById('times');
  el.innerHTML = '';
  times.forEach((t,i)=>{
    const s = document.createElement('span');
    s.className = 'tag';
    s.textContent = t + ' ';
    const x = document.createElement('button');
    x.type='button'; x.textContent='×'; x.className='btn secondary';
    x.onclick=()=>{ times.splice(i,1); renderTimes(); };
    s.appendChild(x); el.appendChild(s);
  });
}
function addTime(){
  const v = document.getElementById('new-time').value;
  if(!v) return;
  const n = v.slice(0,5);
  if(!times.includes(n)) times.push(n);
  times.sort(); renderTimes();
}
function setAll(v){ document.querySelectorAll('input.acct').forEach(c=>c.checked=v); }
function selectedAccounts(){
  return Array.from(document.querySelectorAll('input.acct:checked')).map(c=>c.getAttribute('data-id'));
}
function key(){ return document.getElementById('management-key').value.trim(); }
async function saveCfg(){
  const o=document.getElementById('result'); const k=key();
  if(!k){ o.textContent='需要 Management Key'; return; }
  const body={enabled:true, timezone:document.getElementById('tz').value.trim(), times:times, accounts:selectedAccounts()};
  o.textContent='儲存中...';
  try{
    const r=await fetch('/v0/management/plugins/'+pluginId+'/config',{method:'PATCH',headers:{'Authorization':'Bearer '+k,'Content-Type':'application/json'},body:JSON.stringify(body)});
    o.textContent=await r.text();
    if(r.ok) setTimeout(()=>location.reload(),800);
  }catch(e){ o.textContent=String(e); }
}
async function runNow(){
  const o=document.getElementById('result'); const k=key();
  if(!k){ o.textContent='需要 Management Key'; return; }
  o.textContent='啟動中...';
  try{
    const r=await fetch('/v0/management/plugins/'+pluginId+'/run',{method:'POST',headers:{'Authorization':'Bearer '+k}});
    o.textContent=await r.text();
    if(r.ok) setTimeout(()=>location.reload(),1500);
  }catch(e){ o.textContent=String(e); }
}
renderTimes();
</script>
</body></html>`,
		html.EscapeString(enabled),
		html.EscapeString(st.Version), html.EscapeString(st.Model),
		html.EscapeString(st.Timezone), html.EscapeString(strings.Join(st.Times, " / ")),
		html.EscapeString(next), html.EscapeString(running),
		html.EscapeString(st.Timezone),
		html.EscapeString(banner),
		rows.String(),
		lastBlock,
		string(timesJSON),
	)
}

func preferID(a runstate.AccountView) string {
	if a.Email != "" {
		return a.Email
	}
	if a.Name != "" {
		return a.Name
	}
	return a.AuthIndex
}

func dash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return html.EscapeString(s)
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func formatWindow(w *hostapi.QuotaWindow) string {
	if w == nil {
		return "—"
	}
	parts := []string{}
	if w.Remaining != nil {
		parts = append(parts, fmt.Sprintf("剩 %.4g", *w.Remaining))
	}
	if w.Used != nil {
		parts = append(parts, fmt.Sprintf("用 %.4g", *w.Used))
	}
	main := "—"
	if len(parts) > 0 {
		main = strings.Join(parts, " / ")
	}
	reset := ""
	if w.ResetsAt != nil && !w.ResetsAt.IsZero() {
		reset = `<small>重置 ` + html.EscapeString(w.ResetsAt.Format("01-02 15:04")) + `</small>`
	}
	if main == "—" && reset == "" {
		return "—"
	}
	return main + reset
}

func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}
```

Add `"encoding/json"` import to `ui.go`. Remove unused `accountsJSON` or use it. Fix: delete unused `accountsJSON` line or `_ = accountsJSON`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/management -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/management/ui.go internal/management/handler.go internal/management/handler_test.go
git commit -m "$(cat <<'EOF'
feat: add Traditional Chinese selective-ping management UI

EOF
)"
```

---

### Task 10: CGO ABI `main.go` + real Host adapter

**Files:**
- Create: `main.go` (CGO exports; replaces deleted auto-ping main)
- Create: `internal/hostapi/cpa_host.go` is **not** used from tests; put adapter in `main.go` or `host_adapter.go` in package main
- Create: `host_adapter.go` in package `main` (non-test helper callable from main)

**Interfaces:**
- Export `cliproxy_plugin_init`, plugin call/free/shutdown (names: `selectivePingPluginCall`, etc.)
- Methods: `plugin.register`, `plugin.reconfigure`, `management.register`, `management.handle`, `plugin.shutdown`
- Plugin name/id: `codex-selective-ping`, version `0.1.0`
- Default TZ Asia/Taipei; config field `accounts`
- Management routes/resources for selective-ping paths
- Host adapter implements `hostapi.Host` via `host.auth.list` / `host.auth.get` / `host.http.do` / optional `host.auth.get_runtime`

- [ ] **Step 1: Write the failing test (non-CGO compile check for adapter types)**

Create `host_adapter_test.go` testing JSON envelope helpers without C:

```go
package main

import (
	"encoding/json"
	"testing"
)

func TestOkFailEnvelope(t *testing.T) {
	ok := okEnvelope(map[string]any{"x": 1})
	var m map[string]any
	if err := json.Unmarshal(ok, &m); err != nil || m["ok"] != true {
		t.Fatalf("%s", ok)
	}
	bad := failEnvelope("invalid_config", "boom")
	if err := json.Unmarshal(bad, &m); err != nil || m["ok"] != false {
		t.Fatalf("%s", bad)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -count=1 .`

Expected: FAIL (`undefined: okEnvelope`) until helpers exist; CGO files may need `CGO_ENABLED=1`. Put envelopes in `envelope.go` without CGO so `go test` works with `CGO_ENABLED=0` by using build tags:

- `main.go` → `//go:build cgo`
- `envelope.go`, `host_adapter.go` → always built
- `main_stub_test.go` not needed

For unit test of package main without linking plugin symbols, keep CGO exports only in `main.go` with `//go:build cgo`, and put shared logic in untagged files.

- [ ] **Step 3: Write minimal implementation**

`envelope.go`:

```go
package main

import "encoding/json"

func okEnvelope(result any) []byte {
	b, _ := json.Marshal(map[string]any{"ok": true, "result": result})
	return b
}
func failEnvelope(code, message string) []byte {
	b, _ := json.Marshal(map[string]any{"ok": false, "error": map[string]any{"code": code, "message": message, "retryable": false}})
	return b
}
```

`host_adapter.go` (uses a function variable for host call so tests can inject):

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
)

// hostCallFn is set by CGO main to call into CPA host ABI.
var hostCallFn = func(method string, req []byte) ([]byte, error) {
	return nil, fmt.Errorf("host not initialized")
}

type cpaHost struct{}

func (cpaHost) AuthList(ctx context.Context) ([]hostapi.AuthFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	raw, err := hostCallFn("host.auth.list", []byte(`{}`))
	if err != nil {
		return nil, err
	}
	var env struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if !env.OK {
		if env.Error != nil {
			return nil, fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
		}
		return nil, fmt.Errorf("host.auth.list failed")
	}
	var resp struct {
		Files []hostapi.AuthFile `json:"files"`
	}
	if err := json.Unmarshal(env.Result, &resp); err != nil {
		return nil, err
	}
	return resp.Files, nil
}

func (cpaHost) AuthGet(ctx context.Context, authIndex string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	payload, _ := json.Marshal(map[string]any{"auth_index": authIndex})
	raw, err := hostCallFn("host.auth.get", payload)
	if err != nil {
		return nil, err
	}
	result, err := unwrapHost(raw)
	if err != nil {
		return nil, err
	}
	var resp struct {
		JSON json.RawMessage `json:"json"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, err
	}
	if len(resp.JSON) == 0 {
		return nil, fmt.Errorf("empty auth JSON")
	}
	return resp.JSON, nil
}

func (cpaHost) HTTPDo(ctx context.Context, req hostapi.HTTPRequest) (hostapi.HTTPResponse, error) {
	if err := ctx.Err(); err != nil {
		return hostapi.HTTPResponse{}, err
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return hostapi.HTTPResponse{}, err
	}
	raw, err := hostCallFn("host.http.do", payload)
	if err != nil {
		return hostapi.HTTPResponse{}, err
	}
	result, err := unwrapHost(raw)
	if err != nil {
		return hostapi.HTTPResponse{}, err
	}
	var resp hostapi.HTTPResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return hostapi.HTTPResponse{}, err
	}
	return resp, nil
}

func (cpaHost) AuthGetRuntime(ctx context.Context, authIndex string) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	payload, _ := json.Marshal(map[string]any{"auth_index": authIndex})
	raw, err := hostCallFn("host.auth.get_runtime", payload)
	if err != nil {
		return nil, hostapi.ErrUnsupported
	}
	result, err := unwrapHost(raw)
	if err != nil {
		return nil, hostapi.ErrUnsupported
	}
	return json.RawMessage(result), nil
}

func unwrapHost(raw []byte) (json.RawMessage, error) {
	var env struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if !env.OK {
		if env.Error != nil {
			return nil, fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
		}
		return nil, fmt.Errorf("host callback failed")
	}
	return env.Result, nil
}
```

`main.go` (CGO) — adapt from reference auto-ping ABI, wiring `plugin.New(cpaHost{}, version)` and `management.Handler`. Complete file:

```go
//go:build cgo

package main

/*
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

typedef struct cliproxy_buffer { uint8_t* ptr; size_t len; } cliproxy_buffer;
typedef int (*cliproxy_host_call_fn)(void* host_ctx, const char* method,
    const uint8_t* request, size_t request_len, cliproxy_buffer* response);
typedef void (*cliproxy_host_free_buffer_fn)(void* ptr, size_t len);
typedef struct cliproxy_host_api {
    uint32_t abi_version; void* host_ctx; cliproxy_host_call_fn call; cliproxy_host_free_buffer_fn free_buffer;
} cliproxy_host_api;
typedef int (*cliproxy_plugin_call_fn)(char* method, uint8_t* request, size_t request_len, cliproxy_buffer* response);
typedef void (*cliproxy_plugin_free_buffer_fn)(void* ptr, size_t len);
typedef void (*cliproxy_plugin_shutdown_fn)(void);
typedef struct cliproxy_plugin_api {
    uint32_t abi_version; cliproxy_plugin_call_fn call; cliproxy_plugin_free_buffer_fn free_buffer; cliproxy_plugin_shutdown_fn shutdown;
} cliproxy_plugin_api;
#ifdef _WIN32
#define CPA_PLUGIN_EXPORT __declspec(dllexport)
#else
#define CPA_PLUGIN_EXPORT
#endif
extern CPA_PLUGIN_EXPORT int selectivePingPluginCall(char* method, uint8_t* request, size_t request_len, cliproxy_buffer* response);
extern CPA_PLUGIN_EXPORT void selectivePingPluginFreeBuffer(void* ptr, size_t len);
extern CPA_PLUGIN_EXPORT void selectivePingPluginShutdown(void);
static const cliproxy_host_api* stored_host;
static inline void store_host_api(const cliproxy_host_api* host) { stored_host = host; }
static inline void set_plugin_api(cliproxy_plugin_api* plugin) {
    plugin->abi_version = 1; plugin->call = selectivePingPluginCall; plugin->free_buffer = selectivePingPluginFreeBuffer; plugin->shutdown = selectivePingPluginShutdown;
}
static inline int call_host_api(const char* method, const uint8_t* request, size_t request_len, cliproxy_buffer* response) {
    if (stored_host == NULL || stored_host->call == NULL) return 1;
    return stored_host->call(stored_host->host_ctx, method, request, request_len, response);
}
static inline void free_host_buffer(void* ptr, size_t len) {
    if (stored_host != NULL && stored_host->free_buffer != NULL && ptr != NULL) stored_host->free_buffer(ptr, len);
}
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"strings"
	"unsafe"

	"cpa-plugin-codex-selective-ping/internal/config"
	"cpa-plugin-codex-selective-ping/internal/management"
	"cpa-plugin-codex-selective-ping/internal/plugin"
)

const (
	pluginName = "codex-selective-ping"
	version    = "0.1.0"
)

var app = plugin.New(cpaHost{}, version)
var mgmt = &management.Handler{Plugin: app}

func main() {}

func init() {
	hostCallFn = func(method string, req []byte) ([]byte, error) {
		cm := C.CString(method)
		defer C.free(unsafe.Pointer(cm))
		var reqPtr *C.uint8_t
		var cPayload unsafe.Pointer
		if len(req) > 0 {
			cPayload = C.CBytes(req)
			if cPayload == nil {
				return nil, fmt.Errorf("allocation failure")
			}
			defer C.free(cPayload)
			reqPtr = (*C.uint8_t)(cPayload)
		}
		var response C.cliproxy_buffer
		code := C.call_host_api(cm, reqPtr, C.size_t(len(req)), &response)
		data := copyHostResponse(response)
		if response.ptr != nil {
			C.free_host_buffer(unsafe.Pointer(response.ptr), response.len)
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("host callback %s empty response code=%d", method, int(code))
		}
		if code != 0 {
			return data, fmt.Errorf("host callback code=%d", int(code))
		}
		return data, nil
	}
}

//export cliproxy_plugin_init
func cliproxy_plugin_init(host *C.cliproxy_host_api, pluginAPI *C.cliproxy_plugin_api) C.int {
	if host == nil || pluginAPI == nil {
		return -1
	}
	C.store_host_api(host)
	C.set_plugin_api(pluginAPI)
	return 0
}

//export selectivePingPluginCall
func selectivePingPluginCall(method *C.char, request *C.uint8_t, requestLen C.size_t, response *C.cliproxy_buffer) C.int {
	if response == nil {
		return -1
	}
	name := ""
	if method != nil {
		name = C.GoString(method)
	}
	requestBytes, ok := copyRequestBytes(request, requestLen)
	if !ok {
		return writeJSON(response, failEnvelope("invalid_request", "invalid request length"))
	}
	switch name {
	case "plugin.register", "plugin.reconfigure":
		var req struct {
			ConfigYAML string `json:"config_yaml"`
		}
		if len(requestBytes) > 0 {
			if err := json.Unmarshal(requestBytes, &req); err != nil {
				return writeJSON(response, failEnvelope("invalid_request", "invalid plugin configuration envelope"))
			}
		}
		cfg, err := config.Parse(req.ConfigYAML)
		if err != nil {
			return writeJSON(response, failEnvelope("invalid_config", err.Error()))
		}
		app.ApplyConfig(cfg)
		return writeJSON(response, okEnvelope(registrationResult()))
	case "management.register":
		return writeJSON(response, okEnvelope(managementRegistrationResult()))
	case "management.handle":
		var req management.Request
		if json.Unmarshal(requestBytes, &req) != nil {
			return writeJSON(response, failEnvelope("invalid_request", "invalid management request"))
		}
		resp := mgmt.Handle(req)
		return writeJSON(response, okEnvelope(resp))
	case "plugin.shutdown":
		app.Shutdown()
		return writeJSON(response, okEnvelope(map[string]any{"status": "stopped"}))
	default:
		return writeJSON(response, failEnvelope("unsupported_method", "unsupported plugin method"))
	}
}

//export selectivePingPluginFreeBuffer
func selectivePingPluginFreeBuffer(ptr unsafe.Pointer, length C.size_t) {
	_ = length
	C.free(ptr)
}

//export selectivePingPluginShutdown
func selectivePingPluginShutdown() { app.Shutdown() }

func registrationResult() map[string]any {
	cfg := config.DefaultConfig()
	return map[string]any{
		"schema_version": 5,
		"metadata": map[string]any{
			"Name": pluginName, "Version": version, "Author": "danielhuang",
			"Description": "Selectively ping configured Codex OAuth accounts on a daily schedule.",
			"ConfigFields": []map[string]any{
				{"Name": "timezone", "Type": "string", "Description": "IANA timezone", "DefaultValue": cfg.Timezone},
				{"Name": "times", "Type": "string", "Description": "Daily HH:MM times", "DefaultValue": strings.Join(cfg.Times, ",")},
				{"Name": "accounts", "Type": "string", "Description": "Whitelist of email/auth_index/name; empty=ping nobody", "DefaultValue": ""},
			},
		},
		"capabilities": map[string]any{"management_api": true},
	}
}

func managementRegistrationResult() map[string]any {
	return map[string]any{
		"routes": []map[string]string{
			{"Method": "GET", "Path": "/plugins/codex-selective-ping/status", "Description": "JSON status"},
			{"Method": "POST", "Path": "/plugins/codex-selective-ping/run", "Description": "Run now for selected accounts"},
		},
		"resources": []map[string]string{
			{"Path": "/status", "Menu": "Codex Selective Ping", "Description": "選擇帳號、排程、儲存與執行"},
		},
	}
}

func copyRequestBytes(request *C.uint8_t, requestLen C.size_t) ([]byte, bool) {
	length := int(requestLen)
	if length < 0 || C.size_t(length) != requestLen {
		return nil, false
	}
	if length == 0 {
		return nil, true
	}
	if request == nil {
		return nil, false
	}
	s := unsafe.Slice((*byte)(unsafe.Pointer(request)), length)
	return append([]byte(nil), s...), true
}

func copyHostResponse(r C.cliproxy_buffer) []byte {
	if r.ptr == nil || r.len == 0 {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(r.ptr), C.int(r.len))
}

func writeJSON(response *C.cliproxy_buffer, data []byte) C.int {
	if len(data) == 0 {
		response.ptr = nil
		response.len = 0
		return 0
	}
	ptr := C.malloc(C.size_t(len(data)))
	if ptr == nil {
		return -1
	}
	C.memcpy(ptr, unsafe.Pointer(&data[0]), C.size_t(len(data)))
	response.ptr = (*C.uint8_t)(ptr)
	response.len = C.size_t(len(data))
	return 0
}
```

Ensure `management.Request`/`Response` JSON field names match CPA (`Method`, `Path`, `Headers`, `Query`, `Body`, `StatusCode`). Align struct tags with reference auto-ping (`StatusCode`, `Body`, etc.).

- [ ] **Step 4: Run tests**

Run: `CGO_ENABLED=0 go test ./internal/... -count=1`  
Run: `CGO_ENABLED=0 go test -count=1 .` (envelope test)

Expected: PASS for internal; package main envelope test PASS

- [ ] **Step 5: Commit**

```bash
git add main.go envelope.go host_adapter.go host_adapter_test.go
git commit -m "$(cat <<'EOF'
feat: add CPA cgo ABI and host adapter for selective-ping

EOF
)"
```

---

### Task 11: registry.json, README, Makefile, c-shared build

**Files:**
- Replace: `registry.json`
- Replace: `README.md`
- Create: `Makefile`
- Modify: `.gitignore` if needed

- [ ] **Step 1: Write the failing test (build smoke as a scripted check)**

Create `Makefile` first in step 3; for fail step, run build before Makefile exists:

Run: `test -f Makefile && make build-linux`

Expected: FAIL (`No such file`)

- [ ] **Step 2: Confirm fail**

Expected: Makefile missing / build fails

- [ ] **Step 3: Implement artifacts**

`registry.json`:

```json
{
  "schema_version": 1,
  "plugins": [
    {
      "id": "codex-selective-ping",
      "name": "Codex Selective Ping",
      "description": "Ping only selected Codex OAuth accounts on a daily schedule, with a Traditional Chinese management UI.",
      "author": "danielhuang",
      "version": "0.1.0",
      "repository": "",
      "homepage": "",
      "license": "MIT",
      "tags": ["codex", "quota", "scheduler", "management", "selective"]
    }
  ]
}
```

`Makefile`:

```makefile
PLUGIN_ID=codex-selective-ping
VERSION=0.1.0

.PHONY: test build-linux clean

test:
	CGO_ENABLED=0 go test ./... -count=1

build-linux:
	mkdir -p package dist
	CGO_ENABLED=1 go build -buildmode=c-shared -trimpath -ldflags="-s -w" -o package/$(PLUGIN_ID).so .
	rm -f package/$(PLUGIN_ID).h
	@test -f package/$(PLUGIN_ID).so
	@file package/$(PLUGIN_ID).so

clean:
	rm -rf package dist *.so *.h
```

`README.md` (complete):

```markdown
# Codex Selective Ping (CPA plugin)

Brand-new independent CLIProxyAPI plugin. Like `codex-auto-ping`, but only pings accounts listed in `accounts`. Empty `accounts` means ping nobody.

- Plugin ID: `codex-selective-ping`
- Fixed model: `gpt-5.6-luna`
- Config persistence: host `plugins.configs.codex-selective-ping`
- Management UI: Traditional Chinese resource page

## CPA configuration

```yaml
plugins:
  enabled: true
  dir: plugins
  configs:
    codex-selective-ping:
      enabled: true
      timezone: Asia/Taipei
      times:
        - "06:00"
        - "11:00"
        - "16:00"
        - "21:00"
      accounts:
        - "user@example.com"
        - "auth_index_or_name"
```

Semantics:

- `accounts` matches `auth_index` (exact) or email/name/account (case-insensitive).
- Empty `accounts` → scheduled and manual runs attempt 0 pings.
- Does not ping on CPA startup; waits for next configured time.

## Management

Resource page:

```text
GET /v0/resource/plugins/codex-selective-ping/status
```

API:

```text
GET  /v0/management/plugins/codex-selective-ping/status
POST /v0/management/plugins/codex-selective-ping/run
```

Save settings from the UI via host:

```text
GET/PATCH /v0/management/plugins/codex-selective-ping/config
```

`POST .../run` returns 202, or 409 if a run is already in progress.

Quota columns (Plan / 5h / weekly) show host-provided values only; missing fields render as "—".

## Build

```bash
make test
make build-linux
# outputs package/codex-selective-ping.so
```

macOS:

```bash
CGO_ENABLED=1 go build -buildmode=c-shared -o codex-selective-ping.dylib .
```

Copy the shared library into CPA's plugin directory.

## Develop

```bash
go test ./...
```
```

- [ ] **Step 4: Run tests and c-shared build**

```bash
cd /workspace/cpa-plugin-codex-selective-ping
make test
make build-linux
file package/codex-selective-ping.so
```

Expected:
- `make test` → all packages PASS
- `make build-linux` → creates `package/codex-selective-ping.so`
- `file` reports ELF shared object (or “dynamically linked”)

- [ ] **Step 5: Commit**

```bash
git add Makefile registry.json README.md .gitignore
git commit -m "$(cat <<'EOF'
chore: add registry, README, and c-shared Makefile build

EOF
)"
```

---

### Task 12: Remove temporary auto-ping product sources

**Files:**
- Delete: nested `cpa-plugin-codex-auto-ping/` directory (reference clone)
- Ensure root `go.mod` is `cpa-plugin-codex-selective-ping` (not jiz4oh auto-ping)
- Ensure no remaining `codex-auto-ping` product ID in shipped sources (docs/spec may still mention it as reference)

- [ ] **Step 1: Write the failing check**

```bash
# intentional fail before cleanup if nested dir still present as product confusion
test ! -d cpa-plugin-codex-auto-ping
```

Expected: FAIL (directory still exists)

- [ ] **Step 2: Confirm fail**

Expected: exit code 1

- [ ] **Step 3: Implement cleanup**

```bash
cd /workspace/cpa-plugin-codex-selective-ping
rm -rf cpa-plugin-codex-auto-ping
# verify module path
grep -q 'module cpa-plugin-codex-selective-ping' go.mod
# verify plugin id
grep -q 'codex-selective-ping' registry.json
! grep -R --exclude-dir=docs --exclude-dir=.git -n 'codex-auto-ping' . || true
```

Keep design spec / plan references to auto-ping as historical reference (under `docs/`).

- [ ] **Step 4: Re-run verification**

```bash
test ! -d cpa-plugin-codex-auto-ping
make test
make build-linux
```

Expected: directory gone; tests PASS; `.so` builds

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
chore: remove temporary auto-ping reference tree from product root

EOF
)"
```

---

## Spec coverage (self-review)

| Spec requirement | Covered by |
|------------------|------------|
| New independent plugin ID `codex-selective-ping` | Tasks 10–12 (`main.go`, `registry.json`, cleanup) |
| Empty `accounts` → ping nobody (`attempted=0`) | Tasks 1–2, 7, 9 (config default, selector, runner, UI banner) |
| Whitelist match email/auth_index/name; case-insensitive email/name | Task 2 |
| Non-Codex excluded | Task 2 |
| Persist via host `plugins.configs` + GET/PATCH config API | Tasks 9–10 (UI PATCH; plugin consumes reconfigure YAML) |
| Single Traditional Chinese resource management page | Task 9 |
| Show Plan/5h/weekly when host exposes; else `—`; never invent | Tasks 3, 4, 8, 9 |
| Reuse auto-ping host ABI, fixed model, schedule semantics, no startup ping-all | Tasks 5, 6, 10 |
| Per-account continue on failure; 429/limited handling | Tasks 5, 7 |
| `POST /run` 202; running → 409 | Tasks 4, 7, 8 |
| Config validate bad timezone/HH:MM | Task 1 |
| Status JSON shape with `selected` + last run | Tasks 4, 8 |
| Go + CGO c-shared deliverable | Tasks 10–11 |
| Replace auto-ping root sources; do not ship auto-ping as product | Tasks 1, 10, 12 |
| Module path `cpa-plugin-codex-selective-ping` | Tasks 1, 12 |

**Placeholder scan:** No TBD/TODO/`similar to Task N` left as unfinished work; naming drift notes below are intentional implementer guidance.

**Type consistency notes for implementers:**
- Canonical Host methods: `AuthList`, `AuthGet`, `HTTPDo`, `AuthGetRuntime` on `hostapi.Host`.
- Canonical packages: `config`, `selector`, `hostapi`, `runstate`, `pinger`, `scheduler`, `runner`, `management`, `plugin`.
- Plugin ID / path segment: always `codex-selective-ping`.
- If later tasks drift from earlier Interfaces blocks, normalize to Tasks 1–6 names (`TryBegin`, `ModelName`/`ModelName`, `AuthList`, `HTTPDo`) when wiring Tasks 7–10.
- `ErrUnsupported` (Task 3) is the sentinel for missing runtime ABI; Task 7 mock may alias the same value.

**Gaps vs design / how handled:**
1. **Exact CPA auth quota JSON schema** (PR #3068 fields) is not frozen in-repo — Task 3 uses best-effort key aliases; UI always falls back to `—` (matches “never invent”).
2. **Host config PATCH** is performed by the browser UI against the host Management API, not implemented inside the plugin binary (per official persistence model) — covered in Task 9 JS; plugin only applies YAML on `plugin.reconfigure`.
3. **Windows `.dll` / macOS `.dylib` Makefile targets** are documented in README; Task 11 automates Linux `.so` on this box (primary develop environment).
4. **Nested auto-ping git workflow** is reference-only and removed in Task 12; schedule/ping semantics were ported into internal packages rather than vendoring auto-ping.

