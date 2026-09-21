package pinger

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
)

const (
	CodexURL       = "https://chatgpt.com/backend-api/codex/responses"
	ModelName      = "gpt-5.6-luna"
	DefaultPrompt  = "ping"
	WindowInterval = 5 * time.Hour
	WindowGuard    = 1 * time.Second
	AttemptTimeout = 45 * time.Second
	MaxAttempts    = 3
	RetryBaseDelay = 5 * time.Second
)

var retryDelayFn = func(failedAttempt int) time.Duration {
	if failedAttempt < 1 {
		failedAttempt = 1
	}
	return RetryBaseDelay * time.Duration(1<<uint(failedAttempt-1))
}

type Outcome struct {
	Status     string
	HTTPStatus int
	Retryable  bool
	Error      string
	ResetsAt   time.Time
	Attempts   int
	EligibleAt time.Time
}

type authMaterial struct{ AccessToken, AccountID string }

type codexBody struct {
	Model        string         `json:"model"`
	Instructions string         `json:"instructions"`
	Input        []codexMessage `json:"input"`
	Store        bool           `json:"store"`
	Stream       bool           `json:"stream"`
}
type codexMessage struct {
	Type    string      `json:"type"`
	Role    string      `json:"role"`
	Content []codexPart `json:"content"`
}
type codexPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
type upstreamErrorEnvelope struct {
	Error struct {
		Type            string `json:"type"`
		Message         string `json:"message"`
		ResetsAt        int64  `json:"resets_at"`
		ResetsInSeconds int64  `json:"resets_in_seconds"`
	} `json:"error"`
}

func PingAccount(ctx context.Context, h hostapi.Host, a hostapi.AuthFile, force bool, lastSuccess time.Time) Outcome {
	base := Outcome{}
	if !force && !lastSuccess.IsZero() {
		eligible := lastSuccess.Add(WindowInterval + WindowGuard)
		if time.Now().Before(eligible) {
			wait := time.Until(eligible)
			if deadline, ok := ctx.Deadline(); ok && time.Now().Add(wait).After(deadline) {
				base.Status = "deferred"
				base.Error = "next window is outside this run timeout"
				base.EligibleAt = eligible
				return base
			}
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				base.Status = "failed"
				base.Error = ctx.Err().Error()
				base.EligibleAt = eligible
				return base
			case <-timer.C:
			}
		}
	}
	for attempt := 1; attempt <= MaxAttempts; attempt++ {
		base.Attempts = attempt
		attemptCtx, cancel := context.WithTimeout(ctx, AttemptTimeout)
		out := pingOnce(attemptCtx, h, a)
		cancel()
		base.HTTPStatus = out.HTTPStatus
		base.Error = out.Error
		base.ResetsAt = out.ResetsAt
		if out.Status == "success" {
			base.Status = "success"
			base.EligibleAt = time.Now().Add(WindowInterval + WindowGuard)
			return base
		}
		if out.Status == "limited" {
			base.Status = "limited"
			return base
		}
		if !out.Retryable || attempt == MaxAttempts {
			base.Status = "failed"
			return base
		}
		timer := time.NewTimer(retryDelayFn(attempt))
		select {
		case <-ctx.Done():
			timer.Stop()
			base.Status = "failed"
			base.Error = ctx.Err().Error()
			return base
		case <-timer.C:
		}
	}
	base.Status = "failed"
	base.Error = "retry loop exhausted"
	return base
}

func pingOnce(ctx context.Context, h hostapi.Host, a hostapi.AuthFile) Outcome {
	raw, err := h.AuthGet(ctx, a.AuthIndex)
	if err != nil {
		return Outcome{Status: "failed", Retryable: true, Error: "auth get: " + err.Error()}
	}
	m, err := parseAuthMaterial(raw)
	if err != nil {
		return Outcome{Status: "failed", Retryable: false, Error: err.Error()}
	}
	body, _ := json.Marshal(codexBody{
		Model:        ModelName,
		Instructions: "You are a helpful assistant.",
		Input:        []codexMessage{{Type: "message", Role: "user", Content: []codexPart{{Type: "input_text", Text: DefaultPrompt}}}},
		Store:        false,
		Stream:       true,
	})
	headers := map[string][]string{
		"Accept":        {"text/event-stream"},
		"Authorization": {"Bearer " + m.AccessToken},
		"Content-Type":  {"application/json"},
		"OpenAI-Beta":   {"responses=v1"},
		"originator":    {"codex_cli_rs"},
		"User-Agent":    {"codex_cli_rs/0.76.0"},
	}
	if m.AccountID != "" {
		headers["Chatgpt-Account-Id"] = []string{m.AccountID}
	}
	resp, err := h.HTTPDo(ctx, hostapi.HTTPRequest{Method: "POST", URL: CodexURL, Headers: headers, Body: body})
	if err != nil {
		return Outcome{Status: "failed", Retryable: true, Error: err.Error()}
	}
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return Outcome{Status: "success", HTTPStatus: resp.StatusCode}
	}
	typ, msg, reset := parseUpstreamError(resp.Body)
	if typ == "usage_limit_reached" || resp.StatusCode == 429 {
		if msg == "" {
			if typ == "usage_limit_reached" {
				msg = "usage limit reached"
			} else {
				msg = "upstream HTTP 429"
			}
		}
		return Outcome{Status: "limited", HTTPStatus: resp.StatusCode, Error: msg, ResetsAt: reset}
	}
	if msg == "" {
		msg = fmt.Sprintf("upstream HTTP %d", resp.StatusCode)
	}
	if typ != "" {
		msg = typ + ": " + msg
	}
	return Outcome{Status: "failed", HTTPStatus: resp.StatusCode, Retryable: isRetryableHTTP(resp.StatusCode), Error: msg, ResetsAt: reset}
}

func parseUpstreamError(body []byte) (string, string, time.Time) {
	if len(body) == 0 {
		return "", "", time.Time{}
	}
	var p upstreamErrorEnvelope
	if json.Unmarshal(body, &p) != nil {
		return "", "", time.Time{}
	}
	reset := time.Time{}
	if p.Error.ResetsAt > 0 {
		reset = time.Unix(p.Error.ResetsAt, 0)
	} else if p.Error.ResetsInSeconds > 0 {
		reset = time.Now().Add(time.Duration(p.Error.ResetsInSeconds) * time.Second)
	}
	return strings.TrimSpace(p.Error.Type), strings.TrimSpace(p.Error.Message), reset
}

func isRetryableHTTP(code int) bool {
	switch code {
	case 408, 425, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

func parseAuthMaterial(raw []byte) (authMaterial, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return authMaterial{}, fmt.Errorf("invalid auth JSON")
	}
	token := firstString(root, "access_token", "accessToken", "oauth_access_token", "oauthAccessToken", "token", "id_token", "idToken")
	accountID := firstString(root, "account_id", "chatgpt_account_id", "accountId", "chatgptAccountId")
	for _, key := range []string{"tokens", "credentials", "auth", "oauth", "session"} {
		var nested map[string]json.RawMessage
		if b, ok := root[key]; ok && json.Unmarshal(b, &nested) == nil {
			if token == "" {
				token = firstString(nested, "access_token", "accessToken", "oauth_access_token", "oauthAccessToken", "token", "id_token", "idToken")
			}
			if accountID == "" {
				accountID = firstString(nested, "account_id", "chatgpt_account_id", "accountId", "chatgptAccountId")
			}
		}
	}
	if token == "" {
		return authMaterial{}, fmt.Errorf("missing access token")
	}
	return authMaterial{AccessToken: token, AccountID: accountID}, nil
}

func firstString(m map[string]json.RawMessage, keys ...string) string {
	for _, k := range keys {
		raw, ok := m[k]
		if !ok {
			continue
		}
		var s string
		if json.Unmarshal(raw, &s) == nil && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
