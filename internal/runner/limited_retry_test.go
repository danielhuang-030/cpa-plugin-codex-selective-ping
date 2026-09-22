package runner

import (
	"context"
	"testing"
	"time"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/pinger"
)

func TestPingWithLimitedRetrySuccessAfterOneWait(t *testing.T) {
	calls := 0
	var sleeps []time.Duration
	ping := func(ctx context.Context, h hostapi.Host, a hostapi.AuthFile, force bool, lastSuccess time.Time, model string) pinger.Outcome {
		calls++
		if calls == 1 {
			return pinger.Outcome{Status: "limited", Attempts: 1, Error: "quota"}
		}
		return pinger.Outcome{Status: "success", Attempts: 1}
	}
	sleep := func(ctx context.Context, d time.Duration) error {
		sleeps = append(sleeps, d)
		return nil
	}
	out := pingWithLimitedRetry(context.Background(), nil, hostapi.AuthFile{AuthIndex: "1"}, true, time.Time{}, 2, "", ping, sleep)
	if out.Status != "success" {
		t.Fatalf("status=%q", out.Status)
	}
	if calls != 2 {
		t.Fatalf("calls=%d want 2", calls)
	}
	if len(sleeps) != 1 || sleeps[0] != LimitedRetryInterval {
		t.Fatalf("sleeps=%v want one %v", sleeps, LimitedRetryInterval)
	}
}

func TestPingWithLimitedRetryExhaustsCount(t *testing.T) {
	calls := 0
	var sleeps []time.Duration
	ping := func(ctx context.Context, h hostapi.Host, a hostapi.AuthFile, force bool, lastSuccess time.Time, model string) pinger.Outcome {
		calls++
		return pinger.Outcome{Status: "limited", Attempts: 1, Error: "quota"}
	}
	sleep := func(ctx context.Context, d time.Duration) error {
		sleeps = append(sleeps, d)
		return nil
	}
	out := pingWithLimitedRetry(context.Background(), nil, hostapi.AuthFile{}, true, time.Time{}, 2, "", ping, sleep)
	if out.Status != "limited" {
		t.Fatalf("status=%q", out.Status)
	}
	if calls != 3 { // first + 2 retries
		t.Fatalf("calls=%d want 3", calls)
	}
	if len(sleeps) != 2 {
		t.Fatalf("sleeps=%d want 2", len(sleeps))
	}
}

func TestPingWithLimitedRetryZeroMeansNoRetry(t *testing.T) {
	calls := 0
	ping := func(ctx context.Context, h hostapi.Host, a hostapi.AuthFile, force bool, lastSuccess time.Time, model string) pinger.Outcome {
		calls++
		return pinger.Outcome{Status: "limited", Attempts: 1}
	}
	sleep := func(ctx context.Context, d time.Duration) error {
		t.Fatal("should not sleep")
		return nil
	}
	out := pingWithLimitedRetry(context.Background(), nil, hostapi.AuthFile{}, true, time.Time{}, 0, "", ping, sleep)
	if out.Status != "limited" || calls != 1 {
		t.Fatalf("status=%q calls=%d", out.Status, calls)
	}
}

func TestPingWithLimitedRetryNonLimitedNoOuterRetry(t *testing.T) {
	calls := 0
	ping := func(ctx context.Context, h hostapi.Host, a hostapi.AuthFile, force bool, lastSuccess time.Time, model string) pinger.Outcome {
		calls++
		return pinger.Outcome{Status: "failed", Attempts: 3, Error: "boom", Retryable: true}
	}
	sleep := func(ctx context.Context, d time.Duration) error {
		t.Fatal("should not sleep for non-limited")
		return nil
	}
	out := pingWithLimitedRetry(context.Background(), nil, hostapi.AuthFile{}, true, time.Time{}, 2, "", ping, sleep)
	if out.Status != "failed" || calls != 1 {
		t.Fatalf("status=%q calls=%d", out.Status, calls)
	}
}

func TestPingWithLimitedRetryContextCancelDuringSleep(t *testing.T) {
	calls := 0
	ping := func(ctx context.Context, h hostapi.Host, a hostapi.AuthFile, force bool, lastSuccess time.Time, model string) pinger.Outcome {
		calls++
		return pinger.Outcome{Status: "limited", Attempts: 1}
	}
	ctx, cancel := context.WithCancel(context.Background())
	sleep := func(ctx context.Context, d time.Duration) error {
		cancel()
		return ctx.Err()
	}
	out := pingWithLimitedRetry(ctx, nil, hostapi.AuthFile{}, true, time.Time{}, 2, "", ping, sleep)
	if out.Status != "failed" {
		t.Fatalf("status=%q want failed", out.Status)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
}
