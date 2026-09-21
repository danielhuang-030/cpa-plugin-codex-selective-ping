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
