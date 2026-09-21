package plugin

import (
	"context"
	"sync"

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
	p.Sched = scheduler.New(func(ctx context.Context) {
		p.mu.Lock()
		cfg := p.cfg
		p.mu.Unlock()
		p.Runner.Run(ctx, cfg, false)
	}, p.State.SetNextRun)
	return p
}

func (p *Plugin) ApplyConfig(cfg config.Config) {
	p.mu.Lock()
	p.cfg = cfg
	p.mu.Unlock()
	path := runstate.ResolveStatePath(cfg.StatePath, cfg.DataDir)
	p.State.SetPersistPath(path)
	p.State.SetHistoryLimit(cfg.HistoryLimit)
	p.Sched.Start(cfg)
}

func (p *Plugin) Config() config.Config {
	p.mu.Lock()
	defer p.mu.Unlock()
	cp := p.cfg
	cp.Times = append([]string(nil), p.cfg.Times...)
	cp.Accounts = append([]string(nil), p.cfg.Accounts...)
	return cp
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
