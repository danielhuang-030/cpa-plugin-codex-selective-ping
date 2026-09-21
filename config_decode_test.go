package main

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestParseRegisterConfigJSONDecodesBase64YAML(t *testing.T) {
	yaml := "enabled: true\nschedule_enabled: false\ntimezone: UTC\ntimes:\n  - \"09:00\"\n"
	// CPA json.Marshal of []byte emits standard base64 string.
	payload, err := json.Marshal(map[string]any{
		"config_yaml": []byte(yaml),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Sanity: payload contains base64, not raw YAML key text as top-level readable enabled line inside JSON string value that starts with enabled
	var probe struct {
		ConfigYAML string `json:"config_yaml"`
	}
	if err := json.Unmarshal(payload, &probe); err != nil {
		t.Fatal(err)
	}
	if probe.ConfigYAML == yaml {
		t.Fatal("expected CPA-style base64 encoding of config_yaml")
	}
	if _, err := base64.StdEncoding.DecodeString(probe.ConfigYAML); err != nil {
		t.Fatalf("config_yaml should be base64: %v", err)
	}

	cfg, err := parseRegisterConfigJSON(payload)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled {
		t.Fatalf("schedule_enabled false must apply, got Enabled=%v", cfg.Enabled)
	}
	if cfg.Timezone != "UTC" || len(cfg.Times) != 1 || cfg.Times[0] != "09:00" {
		t.Fatalf("cfg=%#v", cfg)
	}
}

func TestParseRegisterConfigJSONEmpty(t *testing.T) {
	cfg, err := parseRegisterConfigJSON(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Enabled || cfg.Timezone != "Asia/Taipei" {
		t.Fatalf("%#v", cfg)
	}
}
