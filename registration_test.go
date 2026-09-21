package main

import (
	"testing"

	"cpa-plugin-codex-selective-ping/internal/config"
)

func TestConfigFieldsIncludeEnabled(t *testing.T) {
	cfg := config.DefaultConfig()
	fields := configFields(cfg)
	if len(fields) < 1 {
		t.Fatal("expected config fields")
	}
	var enabled map[string]any
	for _, f := range fields {
		if f["Name"] == "enabled" {
			enabled = f
			break
		}
	}
	if enabled == nil {
		t.Fatal("ConfigFields must include enabled")
	}
	if enabled["Type"] != "bool" {
		t.Fatalf("enabled Type=%v want bool", enabled["Type"])
	}
	if enabled["DefaultValue"] != cfg.Enabled {
		t.Fatalf("enabled DefaultValue=%v want %v", enabled["DefaultValue"], cfg.Enabled)
	}
	meta := registrationMeta(cfg)
	md, ok := meta["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata type %T", meta["metadata"])
	}
	cf, ok := md["ConfigFields"].([]map[string]any)
	if !ok || len(cf) == 0 {
		t.Fatalf("ConfigFields missing in registration meta: %#v", md["ConfigFields"])
	}
}

func TestRegistrationMetaIncludesTimezoneTimesAccounts(t *testing.T) {
	names := map[string]bool{}
	for _, f := range configFields(config.DefaultConfig()) {
		names[f["Name"].(string)] = true
	}
	for _, want := range []string{"enabled", "timezone", "times", "accounts"} {
		if !names[want] {
			t.Fatalf("missing field %q", want)
		}
	}
}
