package runner

import (
	"context"
	"time"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/pinger"
)

const LimitedRetryInterval = time.Minute

// pingFn is overridable in tests.
type pingFn func(ctx context.Context, h hostapi.Host, a hostapi.AuthFile, force bool, lastSuccess time.Time) pinger.Outcome

// sleepFn is overridable in tests (production uses ctx-aware sleep).
type sleepFn func(ctx context.Context, d time.Duration) error

func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// pingWithLimitedRetry calls ping once, then on status "limited" waits LimitedRetryInterval
// and retries up to retryCount additional times. Non-limited outcomes stop immediately.
func pingWithLimitedRetry(
	ctx context.Context,
	h hostapi.Host,
	a hostapi.AuthFile,
	force bool,
	lastSuccess time.Time,
	retryCount int,
	ping pingFn,
	sleep sleepFn,
) pinger.Outcome {
	if ping == nil {
		ping = pinger.PingAccount
	}
	if sleep == nil {
		sleep = sleepCtx
	}
	if retryCount < 0 {
		retryCount = 0
	}
	var out pinger.Outcome
	for attempt := 0; attempt <= retryCount; attempt++ {
		if attempt > 0 {
			if err := sleep(ctx, LimitedRetryInterval); err != nil {
				out.Status = "failed"
				out.Error = err.Error()
				return out
			}
		}
		out = ping(ctx, h, a, force, lastSuccess)
		if out.Status != "limited" {
			return out
		}
		if attempt == retryCount {
			return out
		}
	}
	return out
}
