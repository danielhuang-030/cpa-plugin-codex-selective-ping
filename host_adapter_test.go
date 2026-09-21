package main

import (
	"context"
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

func TestAuthListEnrichesNestedRateLimitFromRaw(t *testing.T) {
	old := hostCallFn
	defer func() { hostCallFn = old }()

	hostCallFn = func(method string, req []byte) ([]byte, error) {
		if method != "host.auth.list" {
			t.Fatalf("unexpected method %q", method)
		}
		// Nested rate_limit keys are dropped if AuthList unmarshals straight into AuthFile
		// then remarshals the typed struct — EnrichQuota must see the raw list item.
		result := json.RawMessage(`{
			"files":[{
				"auth_index":"1",
				"name":"alice",
				"provider":"codex",
				"rate_limit":{"five_hour":{"remaining_fraction":0.42}}
			}]
		}`)
		env, _ := json.Marshal(map[string]any{"ok": true, "result": result})
		return env, nil
	}

	files, err := (cpaHost{}).AuthList(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("len=%d", len(files))
	}
	f := files[0]
	if f.FiveHour == nil || f.FiveHour.Remaining == nil || *f.FiveHour.Remaining != 0.42 {
		t.Fatalf("expected FiveHour.Remaining=0.42 from nested rate_limit, got %#v", f.FiveHour)
	}
}
