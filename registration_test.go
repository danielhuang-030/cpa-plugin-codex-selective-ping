package main

import (
	"strings"
	"testing"

	"cpa-plugin-codex-selective-ping/internal/config"
)

func TestConfigFieldsIncludeEnabled(t *testing.T) {
	cfg := config.DefaultConfig()
	fields := configFields(cfg)
	if len(fields) < 1 {
		t.Fatal("expected config fields")
	}
	var schedule map[string]any
	for _, f := range fields {
		if f["Name"] == "schedule_enabled" {
			schedule = f
			break
		}
	}
	if schedule == nil {
		t.Fatal("ConfigFields must include schedule_enabled")
	}
	if schedule["Type"] != "bool" {
		t.Fatalf("schedule_enabled Type=%v want bool", schedule["Type"])
	}
	if schedule["DefaultValue"] != cfg.Enabled {
		t.Fatalf("schedule_enabled DefaultValue=%v want %v", schedule["DefaultValue"], cfg.Enabled)
	}
	desc, _ := schedule["Description"].(string)
	if !strings.Contains(strings.ToLower(desc), "schedule") {
		t.Fatalf("Description should mention schedule, got %q", desc)
	}
	for _, f := range fields {
		if f["Name"] == "enabled" {
			t.Fatal("ConfigFields must not expose host lifecycle key enabled")
		}
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

func TestRegistrationConfigFieldsScheduleEnabled(t *testing.T) {
	names := map[string]bool{}
	for _, f := range configFields(config.DefaultConfig()) {
		names[f["Name"].(string)] = true
	}
	if !names["schedule_enabled"] {
		t.Fatal("ConfigFields name must be schedule_enabled not enabled")
	}
	if names["enabled"] {
		t.Fatal("ConfigFields must not include enabled")
	}
}

func TestRegistrationMetaIncludesTimezoneTimesAccounts(t *testing.T) {
	names := map[string]bool{}
	for _, f := range configFields(config.DefaultConfig()) {
		names[f["Name"].(string)] = true
	}
	for _, want := range []string{"schedule_enabled", "timezone", "times", "accounts"} {
		if !names[want] {
			t.Fatalf("missing field %q", want)
		}
	}
}

func TestRegistrationMetaRequiredByHost(t *testing.T) {
	meta := registrationMeta(config.DefaultConfig())
	md, ok := meta["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata type %T", meta["metadata"])
	}
	for _, key := range []string{"Name", "Version", "Author", "GitHubRepository"} {
		v, ok := md[key].(string)
		if !ok || strings.TrimSpace(v) == "" {
			t.Fatalf("metadata.%s must be non-empty string, got %#v", key, md[key])
		}
	}
	caps, ok := meta["capabilities"].(map[string]any)
	if !ok || caps["management_api"] != true {
		t.Fatalf("capabilities.management_api want true, got %#v", meta["capabilities"])
	}
}
