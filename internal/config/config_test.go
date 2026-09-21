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
