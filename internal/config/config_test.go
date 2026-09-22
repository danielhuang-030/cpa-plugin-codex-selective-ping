package config

import (
	"strings"
	"testing"

	"cpa-plugin-codex-selective-ping/internal/pinger"
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
	cfg, err := Parse(`{"schedule_enabled":false,"timezone":"UTC","times":["07:30"],"accounts":["a"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled || cfg.Timezone != "UTC" || cfg.Times[0] != "07:30" || cfg.Accounts[0] != "a" {
		t.Fatalf("%#v", cfg)
	}
}

func TestParseScheduleEnabledIgnoresHostEnabled(t *testing.T) {
	cfg, err := Parse("enabled: false\ntimezone: UTC\ntimes: [\"06:00\"]")
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Enabled {
		t.Fatalf("host lifecycle enabled must be ignored for schedule; got Enabled=%v", cfg.Enabled)
	}
}

func TestParseScheduleEnabledFalse(t *testing.T) {
	cfg, err := Parse("schedule_enabled: false\ntimezone: UTC\ntimes: [\"06:00\"]")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled {
		t.Fatal("schedule_enabled:false must disable schedule")
	}
}

func TestParseJSONScheduleEnabled(t *testing.T) {
	cfg, err := Parse(`{"schedule_enabled":true,"timezone":"UTC","times":["08:00"],"enabled":false}`)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Enabled {
		t.Fatalf("schedule_enabled:true must win over host enabled:false; got %#v", cfg)
	}
	if cfg.Times[0] != "08:00" {
		t.Fatalf("times=%#v", cfg.Times)
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


func TestParseYAMLDataDirAndStatePath(t *testing.T) {
	raw := `
schedule_enabled: true
timezone: UTC
times: ["06:00"]
data_dir: /var/cpa/data/codex-selective-ping
state_path: /tmp/custom-last-run.json
`
	cfg, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DataDir != "/var/cpa/data/codex-selective-ping" {
		t.Fatalf("DataDir=%q", cfg.DataDir)
	}
	if cfg.StatePath != "/tmp/custom-last-run.json" {
		t.Fatalf("StatePath=%q", cfg.StatePath)
	}
}

func TestParseJSONDataDirAndStatePath(t *testing.T) {
	cfg, err := Parse(`{"schedule_enabled":true,"timezone":"UTC","times":["06:00"],"data_dir":"rel-data","state_path":"rel-state.json"}`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DataDir != "rel-data" || cfg.StatePath != "rel-state.json" {
		t.Fatalf("%#v", cfg)
	}
}

func TestDefaultConfigHistoryLimit(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.HistoryLimit != 60 {
		t.Fatalf("HistoryLimit default=%d want 60", cfg.HistoryLimit)
	}
}

func TestParseJSONHistoryLimit(t *testing.T) {
	cfg, err := Parse(`{"schedule_enabled":true,"timezone":"UTC","times":["06:00"],"history_limit":10}`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HistoryLimit != 10 {
		t.Fatalf("HistoryLimit=%d want 10", cfg.HistoryLimit)
	}
}

func TestParseYAMLHistoryLimit(t *testing.T) {
	raw := `
schedule_enabled: true
timezone: UTC
times: ["06:00"]
history_limit: 25
`
	cfg, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HistoryLimit != 25 {
		t.Fatalf("HistoryLimit=%d want 25", cfg.HistoryLimit)
	}
}

func TestParseHistoryLimitNonPositiveDefaultsTo60(t *testing.T) {
	for _, raw := range []string{
		`{"schedule_enabled":true,"timezone":"UTC","times":["06:00"],"history_limit":0}`,
		`{"schedule_enabled":true,"timezone":"UTC","times":["06:00"],"history_limit":-3}`,
	} {
		cfg, err := Parse(raw)
		if err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		if cfg.HistoryLimit != 60 {
			t.Fatalf("%s: HistoryLimit=%d want 60", raw, cfg.HistoryLimit)
		}
	}
}

func TestParseOmitsHistoryLimitDefaultsTo60(t *testing.T) {
	cfg, err := Parse(`{"schedule_enabled":true,"timezone":"UTC","times":["06:00"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HistoryLimit != 60 {
		t.Fatalf("HistoryLimit=%d want 60", cfg.HistoryLimit)
	}
}


func TestDefaultConfigRetryCount(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.RetryCount != 2 {
		t.Fatalf("RetryCount default=%d want 2", cfg.RetryCount)
	}
}

func TestParseJSONRetryCount(t *testing.T) {
	cfg, err := Parse(`{"schedule_enabled":true,"timezone":"UTC","times":["06:00"],"retry_count":5}`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RetryCount != 5 {
		t.Fatalf("RetryCount=%d want 5", cfg.RetryCount)
	}
}

func TestParseYAMLRetryCount(t *testing.T) {
	raw := `
schedule_enabled: true
timezone: UTC
times: ["06:00"]
retry_count: 1
`
	cfg, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RetryCount != 1 {
		t.Fatalf("RetryCount=%d want 1", cfg.RetryCount)
	}
}

func TestParseRetryCountZeroDisablesOuterRetry(t *testing.T) {
	cfg, err := Parse(`{"schedule_enabled":true,"timezone":"UTC","times":["06:00"],"retry_count":0}`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RetryCount != 0 {
		t.Fatalf("RetryCount=%d want 0", cfg.RetryCount)
	}
}

func TestParseRetryCountNegativeClampsToZero(t *testing.T) {
	cfg, err := Parse(`{"schedule_enabled":true,"timezone":"UTC","times":["06:00"],"retry_count":-2}`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RetryCount != 0 {
		t.Fatalf("RetryCount=%d want 0", cfg.RetryCount)
	}
}

func TestParseOmitsRetryCountDefaultsTo2(t *testing.T) {
	cfg, err := Parse(`{"schedule_enabled":true,"timezone":"UTC","times":["06:00"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RetryCount != 2 {
		t.Fatalf("RetryCount=%d want 2", cfg.RetryCount)
	}
}


func TestParseJSONAccountTimes(t *testing.T) {
	raw := `{
		"schedule_enabled":true,
		"timezone":"UTC",
		"times":["06:00","11:00"],
		"accounts":["alice@example.com","bob@example.com"],
		"account_times":{"bob@example.com":["07:30","19:00"]}
	}`
	cfg, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.AccountTimes) != 1 {
		t.Fatalf("AccountTimes=%#v want 1 entry", cfg.AccountTimes)
	}
	got := cfg.AccountTimes["bob@example.com"]
	if len(got) != 2 || got[0] != "07:30" || got[1] != "19:00" {
		t.Fatalf("bob times=%#v", got)
	}
}

func TestParseJSONOmitsAccountTimes(t *testing.T) {
	cfg, err := Parse(`{"schedule_enabled":true,"timezone":"UTC","times":["06:00"],"accounts":["a"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.AccountTimes) != 0 {
		t.Fatalf("AccountTimes=%#v want empty/nil", cfg.AccountTimes)
	}
}

func TestParseYAMLAccountTimes(t *testing.T) {
	raw := `
schedule_enabled: true
timezone: UTC
times:
  - "06:00"
  - "11:00"
accounts:
  - alice@example.com
  - bob@example.com
account_times:
  bob@example.com:
    - "07:30"
    - "19:00"
`
	cfg, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	got := cfg.AccountTimes["bob@example.com"]
	if len(got) != 2 || got[0] != "07:30" || got[1] != "19:00" {
		t.Fatalf("bob times=%#v AccountTimes=%#v", got, cfg.AccountTimes)
	}
}

func TestEffectiveTimesInheritVsOverride(t *testing.T) {
	cfg := Config{
		Timezone: "UTC",
		Times:    []string{"06:00", "11:00", "16:00", "21:00"},
		Accounts: []string{"alice@example.com", "bob@example.com"},
		AccountTimes: map[string][]string{
			"bob@example.com": {"07:30", "19:00"},
		},
	}
	alice := EffectiveTimes(cfg, "alice@example.com")
	if len(alice) != 4 || alice[0] != "06:00" || alice[3] != "21:00" {
		t.Fatalf("alice inherit=%#v", alice)
	}
	bob := EffectiveTimes(cfg, "bob@example.com")
	if len(bob) != 2 || bob[0] != "07:30" || bob[1] != "19:00" {
		t.Fatalf("bob override=%#v", bob)
	}
}

func TestValidateEmptyAccountTimesListInherits(t *testing.T) {
	cfg, err := Validate(Config{
		Timezone: "UTC",
		Times:    []string{"06:00"},
		Accounts: []string{"alice@example.com", "bob@example.com"},
		AccountTimes: map[string][]string{
			"bob@example.com": {},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.AccountTimes["bob@example.com"]; ok {
		t.Fatalf("empty custom list must delete key (inherit), got %#v", cfg.AccountTimes)
	}
	if EffectiveTimes(cfg, "bob@example.com")[0] != "06:00" {
		t.Fatalf("bob should inherit global after empty prune")
	}
}

func TestValidateRejectsBadAccountTime(t *testing.T) {
	_, err := Validate(Config{
		Timezone: "UTC",
		Times:    []string{"06:00"},
		Accounts: []string{"bob@example.com"},
		AccountTimes: map[string][]string{
			"bob@example.com": {"25:00"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "time") {
		t.Fatalf("err=%v", err)
	}
}

func TestValidatePrunesOrphanAccountTimes(t *testing.T) {
	cfg, err := Validate(Config{
		Timezone: "UTC",
		Times:    []string{"06:00"},
		Accounts: []string{"alice@example.com"},
		AccountTimes: map[string][]string{
			"bob@example.com":   {"07:30"},
			"alice@example.com": {"08:00"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.AccountTimes["bob@example.com"]; ok {
		t.Fatalf("orphan bob must be pruned, got %#v", cfg.AccountTimes)
	}
	got := cfg.AccountTimes["alice@example.com"]
	if len(got) != 1 || got[0] != "08:00" {
		t.Fatalf("alice times=%#v", got)
	}
}

func TestValidateNormalizesAccountTimes(t *testing.T) {
	cfg, err := Validate(Config{
		Timezone: "UTC",
		Times:    []string{"06:00"},
		Accounts: []string{"bob@example.com"},
		AccountTimes: map[string][]string{
			"bob@example.com": {"7:30", "07:30", "19:00"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := cfg.AccountTimes["bob@example.com"]
	if len(got) != 2 || got[0] != "07:30" || got[1] != "19:00" {
		t.Fatalf("normalized=%#v", got)
	}
}

func TestUnionTimesAndAccountsForSlot(t *testing.T) {
	cfg := Config{
		Timezone: "UTC",
		Times:    []string{"06:00", "11:00", "16:00", "21:00"},
		Accounts: []string{"alice@example.com", "bob@example.com"},
		AccountTimes: map[string][]string{
			"bob@example.com": {"07:30", "19:00"},
		},
	}
	union := UnionTimes(cfg)
	wantUnion := []string{"06:00", "07:30", "11:00", "16:00", "19:00", "21:00"}
	if len(union) != len(wantUnion) {
		t.Fatalf("union=%#v want %#v", union, wantUnion)
	}
	for i := range wantUnion {
		if union[i] != wantUnion[i] {
			t.Fatalf("union=%#v want %#v", union, wantUnion)
		}
	}

	at0600 := AccountsForSlot(cfg, "06:00")
	if len(at0600) != 1 || at0600[0] != "alice@example.com" {
		t.Fatalf("06:00 accounts=%#v", at0600)
	}
	at0730 := AccountsForSlot(cfg, "07:30")
	if len(at0730) != 1 || at0730[0] != "bob@example.com" {
		t.Fatalf("07:30 accounts=%#v", at0730)
	}
	at1100 := AccountsForSlot(cfg, "11:00")
	if len(at1100) != 1 || at1100[0] != "alice@example.com" {
		t.Fatalf("11:00 accounts=%#v", at1100)
	}
}

func TestUnionTimesEmptyAllowlistReturnsGlobal(t *testing.T) {
	cfg := Config{
		Timezone: "UTC",
		Times:    []string{"06:00", "21:00"},
		Accounts: []string{},
	}
	union := UnionTimes(cfg)
	if len(union) != 2 || union[0] != "06:00" || union[1] != "21:00" {
		t.Fatalf("union=%#v want global Times", union)
	}
}

func TestAccountsForSlotNormalizesHHMM(t *testing.T) {
	cfg := Config{
		Timezone: "UTC",
		Times:    []string{"06:00"},
		Accounts: []string{"alice@example.com"},
	}
	got := AccountsForSlot(cfg, "6:00")
	if len(got) != 1 || got[0] != "alice@example.com" {
		t.Fatalf("AccountsForSlot with unpadded hhmm=%#v", got)
	}
}


func TestParseModel(t *testing.T) {
	cfg, err := Parse(`{"schedule_enabled":true,"timezone":"Asia/Taipei","times":["06:00"],"model":"gpt-5"}`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "gpt-5" {
		t.Fatalf("model=%q", cfg.Model)
	}
}

func TestParseYAMLModel(t *testing.T) {
	raw := `
schedule_enabled: true
timezone: Asia/Taipei
times: ["06:00"]
model: gpt-5
`
	cfg, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "gpt-5" {
		t.Fatalf("model=%q", cfg.Model)
	}
}

func TestParseOmitsModelLeavesEmpty(t *testing.T) {
	cfg, err := Parse(`{"schedule_enabled":true,"timezone":"UTC","times":["06:00"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "" {
		t.Fatalf("Model=%q want empty", cfg.Model)
	}
}

func TestEffectiveModelEmptyFallsBack(t *testing.T) {
	if got := EffectiveModel(Config{}, pinger.ModelName); got != pinger.ModelName {
		t.Fatalf("got %q", got)
	}
}

func TestEffectiveModelWhitespaceFallsBack(t *testing.T) {
	if got := EffectiveModel(Config{Model: "  \t"}, pinger.ModelName); got != pinger.ModelName {
		t.Fatalf("got %q", got)
	}
}

func TestEffectiveModelUsesConfig(t *testing.T) {
	if got := EffectiveModel(Config{Model: "gpt-5"}, pinger.ModelName); got != "gpt-5" {
		t.Fatalf("got %q", got)
	}
}

func TestDefaultConfigModelEmpty(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Model != "" {
		t.Fatalf("Model default=%q want empty", cfg.Model)
	}
}
