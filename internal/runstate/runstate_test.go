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
