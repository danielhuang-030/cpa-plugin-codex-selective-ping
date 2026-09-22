package scheduler

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cpa-plugin-codex-selective-ping/internal/config"
)

func TestNextRunPicksSoonestFutureSlot(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 9, 21, 10, 0, 0, 0, loc)
	next := NextRun(now, loc, []string{"06:00", "11:00", "16:00"})
	want := time.Date(2026, 9, 21, 11, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("got %s want %s", next, want)
	}
}

func TestNextRunRollsToTomorrow(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 9, 21, 22, 0, 0, 0, loc)
	next := NextRun(now, loc, []string{"06:00", "11:00"})
	want := time.Date(2026, 9, 22, 6, 0, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("got %s want %s", next, want)
	}
}

func TestNextRunFromConfigUsesUnionTimes(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 9, 22, 6, 30, 0, 0, loc)
	cfg := config.Config{
		Enabled:  true,
		Timezone: "UTC",
		Times:    []string{"06:00", "11:00", "16:00", "21:00"},
		Accounts: []string{"alice@example.com", "bob@example.com"},
		AccountTimes: map[string][]string{
			"bob@example.com": {"07:30", "19:00"},
		},
	}
	next := NextRunFromConfig(now, loc, cfg)
	want := time.Date(2026, 9, 22, 7, 30, 0, 0, loc)
	if !next.Equal(want) {
		t.Fatalf("got %s want %s (union should include bob 07:30)", next, want)
	}
}

func TestOnFireReceivesScheduledInstant(t *testing.T) {
	loc := time.UTC
	slot := time.Date(2026, 9, 22, 7, 30, 0, 0, loc)

	var firedAt time.Time
	var fired atomic.Bool
	done := make(chan struct{})
	var once sync.Once
	var pastSlot atomic.Bool

	s := New(func(ctx context.Context, at time.Time) {
		pastSlot.Store(true)
		once.Do(func() {
			firedAt = at
			fired.Store(true)
			close(done)
		})
	}, nil)

	s.now = func() time.Time {
		if pastSlot.Load() {
			return slot.Add(time.Minute)
		}
		return slot.Add(-80 * time.Millisecond)
	}

	cfg := config.Config{
		Enabled:  true,
		Timezone: "UTC",
		Times:    []string{"07:30"},
		Accounts: []string{"bob@example.com"},
	}
	s.Start(cfg)
	defer s.Stop()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for onFire")
	}

	if !fired.Load() {
		t.Fatal("onFire was not invoked")
	}
	if !firedAt.Equal(slot) {
		t.Fatalf("onFire at %s want %s", firedAt, slot)
	}
}

func TestStartUsesUnionTimesForNext(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 9, 22, 6, 30, 0, 0, loc)

	s := New(func(context.Context, time.Time) {}, nil)
	s.now = func() time.Time { return now }

	cfg := config.Config{
		Enabled:  true,
		Timezone: "UTC",
		Times:    []string{"06:00", "11:00"},
		Accounts: []string{"alice@example.com", "bob@example.com"},
		AccountTimes: map[string][]string{
			"bob@example.com": {"07:30", "19:00"},
		},
	}
	s.Start(cfg)
	defer s.Stop()

	// Allow the loop to publish next.
	deadline := time.Now().Add(time.Second)
	var got time.Time
	for time.Now().Before(deadline) {
		got = s.Next()
		if !got.IsZero() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	want := time.Date(2026, 9, 22, 7, 30, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("Next()=%s want %s (Start should schedule UnionTimes, not raw Times)", got, want)
	}
}
