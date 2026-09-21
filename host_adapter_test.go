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
