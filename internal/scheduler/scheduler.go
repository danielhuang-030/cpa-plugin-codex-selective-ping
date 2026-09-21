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
	onFire  func(context.Context)
	setNext func(time.Time)
}

func New(onFire func(context.Context), setNext func(time.Time)) *Scheduler {
	if setNext == nil {
		setNext = func(time.Time) {}
	}
	return &Scheduler{onFire: onFire, setNext: setNext}
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
	go s.loop(ctx, loc, append([]string(nil), cfg.Times...))
}

func (s *Scheduler) loop(ctx context.Context, loc *time.Location, times []string) {
	for {
		next := NextRun(time.Now().In(loc), loc, times)
		s.mu.Lock()
		s.next = next
		s.mu.Unlock()
		s.setNext(next)
		timer := time.NewTimer(time.Until(next))
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
				s.onFire(ctx)
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
