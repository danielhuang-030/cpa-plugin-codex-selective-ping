package management

import (
	"context"
	"encoding/json"
	"strings"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/pinger"
	"cpa-plugin-codex-selective-ping/internal/plugin"
	"cpa-plugin-codex-selective-ping/internal/runstate"
	"cpa-plugin-codex-selective-ping/internal/selector"
)

type Request struct {
	Method  string              `json:"Method"`
	Path    string              `json:"Path"`
	Headers map[string][]string `json:"Headers,omitempty"`
	Query   map[string][]string `json:"Query,omitempty"`
	Body    []byte              `json:"Body,omitempty"`
}

type Response struct {
	StatusCode int                 `json:"StatusCode"`
	Headers    map[string][]string `json:"Headers,omitempty"`
	Body       []byte              `json:"Body,omitempty"`
}

type StatusResponse struct {
	Enabled          bool                   `json:"schedule_enabled"`
	Version          string                 `json:"version"`
	Model            string                 `json:"model"`
	Timezone         string                 `json:"timezone"`
	Times            []string               `json:"times"`
	AccountsConfig   []string               `json:"accounts_config"`
	WindowSeconds    int64                  `json:"window_seconds"`
	GuardSeconds     int64                  `json:"guard_seconds"`
	MaxAttempts      int                    `json:"max_attempts"`
	RetryBaseSeconds int64                  `json:"retry_base_seconds"`
	NextRun          interface{}            `json:"next_run,omitempty"`
	Running          bool                   `json:"running"`
	LastRun          *runstate.Summary      `json:"last_run,omitempty"`
	Accounts         []runstate.AccountView `json:"accounts,omitempty"`
}

type Handler struct {
	Plugin *plugin.Plugin
}

func (h *Handler) Handle(req Request) Response {
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	path := strings.TrimSpace(req.Path)
	switch {
	case method == "GET" && strings.Contains(path, "/v0/resource/plugins/") && strings.HasSuffix(path, "/status"):
		lang := resolveRequestLang(req)
		body := []byte(RenderStatusPage(h.status(), lang))
		return Response{StatusCode: 200, Headers: map[string][]string{"content-type": {"text/html; charset=utf-8"}, "cache-control": {"no-store"}}, Body: body}
	case method == "GET" && strings.HasSuffix(path, "/plugins/codex-selective-ping/status"):
		body, _ := json.Marshal(h.status())
		return Response{StatusCode: 200, Headers: map[string][]string{"content-type": {"application/json; charset=utf-8"}, "cache-control": {"no-store"}}, Body: body}
	case method == "POST" && strings.HasSuffix(path, "/plugins/codex-selective-ping/run"):
		if !h.Plugin.StartManualRun() {
			body, _ := json.Marshal(map[string]any{"accepted": false, "running": true, "message": "a run is already in progress"})
			return Response{StatusCode: 409, Headers: map[string][]string{"content-type": {"application/json; charset=utf-8"}}, Body: body}
		}
		body, _ := json.Marshal(map[string]any{"accepted": true, "running": true, "force": true})
		return Response{StatusCode: 202, Headers: map[string][]string{"content-type": {"application/json; charset=utf-8"}}, Body: body}
	default:
		body, _ := json.Marshal(map[string]string{"error": "not found"})
		return Response{StatusCode: 404, Headers: map[string][]string{"content-type": {"application/json; charset=utf-8"}}, Body: body}
	}
}

func (h *Handler) status() StatusResponse {
	cfg := h.Plugin.Config()
	ctx := context.Background()
	files, _ := h.Plugin.Host.AuthList(ctx)
	enriched := make([]hostapi.AuthFile, 0, len(files))
	for _, f := range files {
		if !selector.IsCodex(f) {
			continue
		}
		enriched = append(enriched, hostapi.EnrichFromHost(ctx, h.Plugin.Host, f))
	}
	snap := h.Plugin.State.Snapshot(cfg.Accounts, enriched, h.Plugin.Sched.Next())
	return StatusResponse{
		Enabled: cfg.Enabled, Version: h.Plugin.Version, Model: pinger.ModelName,
		Timezone: cfg.Timezone, Times: cfg.Times, AccountsConfig: cfg.Accounts,
		WindowSeconds: int64(pinger.WindowInterval.Seconds()), GuardSeconds: int64(pinger.WindowGuard.Seconds()),
		MaxAttempts: pinger.MaxAttempts, RetryBaseSeconds: int64(pinger.RetryBaseDelay.Seconds()),
		NextRun: snap.NextRun, Running: snap.Running, LastRun: snap.LastRun, Accounts: snap.Accounts,
	}
}

func resolveRequestLang(req Request) Lang {
	q := firstValue(req.Query, "lang")
	if q != "" {
		return ResolveLang(q, "")
	}
	// Honor CPA language if host forwarded it (do not require client localStorage for first paint).
	if cpa := firstHeader(req.Headers, "X-Cli-Proxy-Language", "cli-proxy-language"); cpa != "" {
		return NormalizeLang(cpa)
	}
	return ResolveLang("", firstHeader(req.Headers, "Accept-Language"))
}

func firstValue(m map[string][]string, key string) string {
	if m == nil {
		return ""
	}
	if vs, ok := m[key]; ok && len(vs) > 0 {
		return strings.TrimSpace(vs[0])
	}
	// case-insensitive fallback for query keys
	for k, vs := range m {
		if strings.EqualFold(k, key) && len(vs) > 0 {
			return strings.TrimSpace(vs[0])
		}
	}
	return ""
}

func firstHeader(h map[string][]string, keys ...string) string {
	if h == nil {
		return ""
	}
	for _, key := range keys {
		if vs, ok := h[key]; ok && len(vs) > 0 {
			return strings.TrimSpace(vs[0])
		}
		for k, vs := range h {
			if strings.EqualFold(k, key) && len(vs) > 0 {
				return strings.TrimSpace(vs[0])
			}
		}
	}
	return ""
}
