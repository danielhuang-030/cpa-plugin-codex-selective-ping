package scheduler

import (
	"context"
	"sync"
	"time"

	"cpa-plugin-codex-selective-ping/internal/config"
)

type Scheduler struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	next    time.Time
	onFire  func(context.Context, time.Time)
	setNext func(time.Time)
	now     func() time.Time // nil => time.Now; tests may override
}

func New(onFire func(context.Context, time.Time), setNext func(time.Time)) *Scheduler {
	if setNext == nil {
		setNext = func(time.Time) {}
	}
	return &Scheduler{onFire: onFire, setNext: setNext}
}

func (s *Scheduler) currentTime() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

func NextRun(now time.Time, loc *time.Location, times []string) time.Time {
	best := time.Time{}
	for _, v := range times {
		h, m, err := config.ParseClock(v)
		if err != nil {
			continue
		}
		c := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, loc)
		if !c.After(now) {
			c = c.AddDate(0, 0, 1)
		}
		if best.IsZero() || c.Before(best) {
			best = c
		}
	}
	return best
}

// NextRunFromConfig returns the next fire instant using the union of effective times.
func NextRunFromConfig(now time.Time, loc *time.Location, cfg config.Config) time.Time {
	return NextRun(now, loc, config.UnionTimes(cfg))
}

func (s *Scheduler) Start(cfg config.Config) {
	s.Stop()
	s.mu.Lock()
	defer s.mu.Unlock()
	if !cfg.Enabled {
		s.next = time.Time{}
		s.setNext(time.Time{})
		return
	}
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.loop(ctx, loc, config.UnionTimes(cfg))
}

func (s *Scheduler) loop(ctx context.Context, loc *time.Location, times []string) {
	for {
		now := s.currentTime().In(loc)
		next := NextRun(now, loc, times)
		s.mu.Lock()
		s.next = next
		s.mu.Unlock()
		s.setNext(next)
		delay := next.Sub(s.currentTime())
		if delay < 0 {
			delay = 0
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case <-timer.C:
			if s.onFire != nil {
				s.onFire(ctx, next)
			}
		}
	}
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.next = time.Time{}
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.setNext(time.Time{})
}

func (s *Scheduler) Next() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.next
}
