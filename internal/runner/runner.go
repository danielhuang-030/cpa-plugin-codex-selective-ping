package runner

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"cpa-plugin-codex-selective-ping/internal/config"
	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/pinger"
	"cpa-plugin-codex-selective-ping/internal/runstate"
	"cpa-plugin-codex-selective-ping/internal/selector"
)

const RunTimeout = 20 * time.Minute

type Runner struct {
	Host  hostapi.Host
	State *runstate.State
}

func (r *Runner) Run(parent context.Context, cfg config.Config, force bool) (runstate.Summary, bool) {
	if !r.State.TryBegin() {
		return runstate.Summary{}, false
	}
	return r.RunClaimed(parent, cfg, force), true
}

// RunClaimed assumes State.TryBegin already succeeded.
func (r *Runner) RunClaimed(parent context.Context, cfg config.Config, force bool) runstate.Summary {
	mode := "scheduled"
	if force {
		mode = "force"
	}
	summary := runstate.Summary{At: time.Now(), Mode: mode}
	defer func() { r.State.End(summary) }()

	ctx, cancel := context.WithTimeout(parent, RunTimeout)
	defer cancel()

	if len(cfg.Accounts) == 0 {
		summary.Message = "no accounts selected; skipped ping (empty accounts whitelist)"
		summary.Attempted = 0
		return summary
	}

	files, err := r.Host.AuthList(ctx)
	if err != nil {
		summary.Error = "auth list failed: " + err.Error()
		return summary
	}
	files = enrichAll(ctx, r.Host, files)

	codexCount := 0
	for _, f := range files {
		if selector.IsCodex(f) {
			codexCount++
		}
	}
	summary.Total = codexCount

	selected := selector.Select(files, cfg.Accounts)
	if len(selected) == 0 {
		summary.Message = "no matching Codex accounts for configured whitelist"
		summary.Attempted = 0
		return summary
	}

	targets := make([]hostapi.AuthFile, 0, len(selected))
	for _, a := range selected {
		if a.Disabled {
			res := runstate.AccountResult{AuthIndex: a.AuthIndex, Name: nameOf(a), Email: a.Email, Unavailable: a.Unavailable, Status: "skipped_disabled"}
			summary.Skipped++
			summary.Accounts = append(summary.Accounts, res)
			continue
		}
		if a.AuthIndex == "" {
			res := runstate.AccountResult{Name: nameOf(a), Email: a.Email, Unavailable: a.Unavailable, Status: "skipped_missing_auth_index", Error: "missing auth_index"}
			summary.Skipped++
			summary.Accounts = append(summary.Accounts, res)
			continue
		}
		targets = append(targets, a)
	}

	results := make(chan runstate.AccountResult, len(targets))
	var wg sync.WaitGroup
	for _, a := range targets {
		wg.Add(1)
		go func(auth hostapi.AuthFile) {
			defer wg.Done()
			mem := r.State.GetAccount(auth.AuthIndex)
			out := pinger.PingAccount(ctx, r.Host, auth, force, mem.LastSuccess, mem.ResetsAt)
			res := runstate.AccountResult{
				AuthIndex: auth.AuthIndex, Name: nameOf(auth), Email: auth.Email, Unavailable: auth.Unavailable,
				Status: out.Status, Attempts: out.Attempts, HTTPStatus: out.HTTPStatus, Error: out.Error,
			}
			if !out.EligibleAt.IsZero() {
				t := out.EligibleAt
				res.EligibleAt = &t
			}
			if !out.ResetsAt.IsZero() {
				t := out.ResetsAt
				res.ResetsAt = &t
			}
			results <- res
		}(a)
	}
	go func() { wg.Wait(); close(results) }()

	firstErr := ""
	for res := range results {
		summary.Accounts = append(summary.Accounts, res)
		switch res.Status {
		case "success":
			summary.Attempted++
			summary.Succeeded++
		case "limited":
			if res.Attempts > 0 {
				summary.Attempted++
			}
			summary.Limited++
		case "deferred":
			summary.Skipped++
		default:
			if res.Attempts > 0 {
				summary.Attempted++
			}
			summary.Failed++
			if firstErr == "" {
				firstErr = res.Error
			}
		}
	}
	sort.Slice(summary.Accounts, func(i, j int) bool {
		return summary.Accounts[i].Name < summary.Accounts[j].Name
	})
	if summary.Failed > 0 {
		summary.Error = firstErr
	}
	return summary
}

func enrichAll(ctx context.Context, h hostapi.Host, files []hostapi.AuthFile) []hostapi.AuthFile {
	out := make([]hostapi.AuthFile, len(files))
	for i, f := range files {
		var runtime json.RawMessage
		if raw, err := h.AuthGetRuntime(ctx, f.AuthIndex); err == nil {
			runtime = raw
		}
		extra, _ := json.Marshal(f)
		out[i] = hostapi.EnrichQuota(f, extra, runtime)
	}
	return out
}

func nameOf(a hostapi.AuthFile) string {
	if a.Name != "" {
		return a.Name
	}
	if a.ID != "" {
		return a.ID
	}
	if a.Email != "" {
		return a.Email
	}
	return a.AuthIndex
}
