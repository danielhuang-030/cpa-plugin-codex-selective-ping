package plugin

import (
	"context"
	"sync"
	"time"

	"cpa-plugin-codex-selective-ping/internal/config"
	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/runner"
	"cpa-plugin-codex-selective-ping/internal/runstate"
	"cpa-plugin-codex-selective-ping/internal/scheduler"
)

const PluginID = "codex-selective-ping"

type Plugin struct {
	Host    hostapi.Host
	Version string
	State   *runstate.State
	Runner  *runner.Runner
	Sched   *scheduler.Scheduler

	mu  sync.Mutex
	cfg config.Config
}

func New(h hostapi.Host, version string) *Plugin {
	p := &Plugin{Host: h, Version: version, State: runstate.New(), cfg: config.DefaultConfig()}
	p.Runner = &runner.Runner{Host: h, State: p.State}
	p.Sched = scheduler.New(func(ctx context.Context, at time.Time) {
		p.mu.Lock()
		cfg := cloneConfig(p.cfg)
		p.mu.Unlock()
		runCfg, ok := configForScheduledFire(cfg, at)
		if !ok {
			return
		}
		p.Runner.Run(ctx, runCfg, false)
	}, p.State.SetNextRun)
	return p
}

func (p *Plugin) ApplyConfig(cfg config.Config) {
	cloned := cloneConfig(cfg)
	p.mu.Lock()
	p.cfg = cloned
	p.mu.Unlock()
	path := runstate.ResolveStatePath(cloned.StatePath, cloned.DataDir)
	p.State.SetPersistPath(path)
	p.State.SetHistoryLimit(cloned.HistoryLimit)
	p.Sched.Start(cloned)
}

func (p *Plugin) Config() config.Config {
	p.mu.Lock()
	defer p.mu.Unlock()
	return cloneConfig(p.cfg)
}

func (p *Plugin) StartManualRun() bool {
	if !p.State.TryBegin() {
		return false
	}
	cfg := p.Config()
	go p.Runner.RunClaimed(context.Background(), cfg, true)
	return true
}

func (p *Plugin) Shutdown() {
	p.Sched.Stop()
}

// cloneConfig deep-copies slices and AccountTimes so callers cannot mutate shared state.
func cloneConfig(cfg config.Config) config.Config {
	cp := cfg
	cp.Times = append([]string(nil), cfg.Times...)
	cp.Accounts = append([]string(nil), cfg.Accounts...)
	if cfg.AccountTimes != nil {
		cp.AccountTimes = make(map[string][]string, len(cfg.AccountTimes))
		for k, v := range cfg.AccountTimes {
			cp.AccountTimes[k] = append([]string(nil), v...)
		}
	}
	return cp
}

// configForScheduledFire returns a config copy with Accounts filtered to the fire slot.
// ok is false when timezone cannot be loaded (skip the scheduled run), matching scheduler.Start.
func configForScheduledFire(cfg config.Config, at time.Time) (config.Config, bool) {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return config.Config{}, false
	}
	out := cloneConfig(cfg)
	slot := at.In(loc).Format("15:04")
	out.Accounts = config.AccountsForSlot(out, slot)
	return out, true
}
