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
	if got.FiveHour == nil || got.FiveHour.Remaining == nil || *got.FiveHour.Remaining != 0.5 {
		t.Fatalf("five_hour remaining_fraction=%#v", got.FiveHour)
	}
	_ = time.Now()
}
