package scheduler

import (
	"testing"
	"time"
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
