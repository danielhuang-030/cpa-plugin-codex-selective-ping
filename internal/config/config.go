package config

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

var defaultTimes = []string{"06:00", "11:00", "16:00", "21:00"}

type Config struct {
	Enabled      bool                `json:"schedule_enabled"`
	Timezone     string              `json:"timezone"`
	Times        []string            `json:"times"`
	Accounts     []string            `json:"accounts"`
	AccountTimes map[string][]string `json:"account_times,omitempty"`
	DataDir      string              `json:"data_dir,omitempty"`
	StatePath    string              `json:"state_path,omitempty"`
	HistoryLimit int                 `json:"history_limit,omitempty"`
	RetryCount   int                 `json:"retry_count,omitempty"`
	Model        string              `json:"model,omitempty"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:      true,
		Timezone:     "Asia/Taipei",
		Times:        append([]string(nil), defaultTimes...),
		Accounts:     []string{},
		HistoryLimit: 60,
		RetryCount:   2,
	}
}

func Parse(raw string) (Config, error) {
	cfg := DefaultConfig()
	text := strings.TrimSpace(raw)
	if text == "" {
		return Validate(cfg)
	}
	if strings.HasPrefix(text, "{") {
		var p struct {
			ScheduleEnabled *bool               `json:"schedule_enabled"`
			Timezone        string              `json:"timezone"`
			Times           []string            `json:"times"`
			Accounts        []string            `json:"accounts"`
			AccountTimes    map[string][]string `json:"account_times"`
			DataDir         string              `json:"data_dir"`
			StatePath       string              `json:"state_path"`
			HistoryLimit    *int                `json:"history_limit"`
			RetryCount      *int                `json:"retry_count"`
			Model           string              `json:"model"`
		}
		if err := json.Unmarshal([]byte(text), &p); err != nil {
			return Config{}, fmt.Errorf("invalid JSON config: %w", err)
		}
		// Ignore host lifecycle "enabled"; only schedule_enabled controls the schedule.
		if p.ScheduleEnabled != nil {
			cfg.Enabled = *p.ScheduleEnabled
		}
		if strings.TrimSpace(p.Timezone) != "" {
			cfg.Timezone = strings.TrimSpace(p.Timezone)
		}
		if p.Times != nil {
			cfg.Times = p.Times
		}
		if p.Accounts != nil {
			cfg.Accounts = p.Accounts
		}
		if p.AccountTimes != nil {
			cfg.AccountTimes = p.AccountTimes
		}
		if strings.TrimSpace(p.DataDir) != "" {
			cfg.DataDir = strings.TrimSpace(p.DataDir)
		}
		if strings.TrimSpace(p.StatePath) != "" {
			cfg.StatePath = strings.TrimSpace(p.StatePath)
		}
		if p.HistoryLimit != nil {
			cfg.HistoryLimit = *p.HistoryLimit
		}
		if p.RetryCount != nil {
			cfg.RetryCount = *p.RetryCount
		}
		if p.Model != "" {
			cfg.Model = p.Model
		}
		return Validate(cfg)
	}
	return parseYAMLSubset(text, cfg)
}

func parseYAMLSubset(text string, cfg Config) (Config, error) {
	lines := strings.Split(text, "\n")
	mode := ""
	var times []string
	var accounts []string
	accountTimes := map[string][]string{}
	accountTimesKey := ""
	inAccountTimes := false
	for _, rawLine := range lines {
		trimmedComment := strings.SplitN(rawLine, "#", 2)[0]
		line := strings.TrimSpace(trimmedComment)
		if line == "" {
			continue
		}
		indent := len(trimmedComment) - len(strings.TrimLeft(trimmedComment, " \t"))
		if strings.HasPrefix(line, "-") && (mode == "times" || mode == "accounts" || mode == "account_times_list") {
			item := unquote(strings.TrimSpace(strings.TrimPrefix(line, "-")))
			if mode == "times" {
				times = append(times, item)
			} else if mode == "accounts" {
				accounts = append(accounts, item)
			} else if mode == "account_times_list" && accountTimesKey != "" {
				accountTimes[accountTimesKey] = append(accountTimes[accountTimesKey], item)
			}
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if inAccountTimes && indent > 0 && key != "account_times" {
			accountTimesKey = unquote(key)
			mode = "account_times_list"
			if value != "" {
				accountTimes[accountTimesKey] = parseInlineList(value)
				mode = ""
			} else if _, ok := accountTimes[accountTimesKey]; !ok {
				accountTimes[accountTimesKey] = nil
			}
			continue
		}
		mode = ""
		accountTimesKey = ""
		inAccountTimes = false
		switch key {
		case "schedule_enabled":
			if value != "" {
				b, err := strconv.ParseBool(unquote(value))
				if err != nil {
					return Config{}, fmt.Errorf("schedule_enabled must be true or false")
				}
				cfg.Enabled = b
			}
		case "enabled":
			// Host lifecycle flag injected by CPA — ignore for schedule purposes.
		case "timezone":
			if value != "" {
				cfg.Timezone = unquote(value)
			}
		case "times":
			mode = "times"
			if value != "" {
				times = parseInlineList(value)
				mode = ""
			}
		case "accounts":
			mode = "accounts"
			if value != "" {
				accounts = parseInlineList(value)
				mode = ""
			}
		case "data_dir":
			if value != "" {
				cfg.DataDir = unquote(value)
			}
		case "state_path":
			if value != "" {
				cfg.StatePath = unquote(value)
			}
		case "history_limit":
			if value != "" {
				n, err := strconv.Atoi(unquote(value))
				if err != nil {
					return Config{}, fmt.Errorf("history_limit must be an integer")
				}
				cfg.HistoryLimit = n
			}
		case "retry_count":
			if value != "" {
				n, err := strconv.Atoi(unquote(value))
				if err != nil {
					return Config{}, fmt.Errorf("retry_count must be an integer")
				}
				cfg.RetryCount = n
			}
		case "model":
			if value != "" {
				cfg.Model = unquote(value)
			}
		case "account_times":
			inAccountTimes = true
			mode = "account_times"
			if value != "" {
				// Inline map forms are not supported by this YAML subset.
				inAccountTimes = false
				mode = ""
			}
		}
	}
	if times != nil {
		cfg.Times = times
	}
	if accounts != nil {
		cfg.Accounts = accounts
	}
	if len(accountTimes) > 0 {
		cfg.AccountTimes = accountTimes
	}
	return Validate(cfg)
}

func Validate(cfg Config) (Config, error) {
	cfg.Timezone = strings.TrimSpace(cfg.Timezone)
	if cfg.Timezone == "" {
		cfg.Timezone = "Asia/Taipei"
	}
	if _, err := time.LoadLocation(cfg.Timezone); err != nil {
		return Config{}, fmt.Errorf("invalid timezone %q", cfg.Timezone)
	}
	if len(cfg.Times) == 0 {
		return Config{}, fmt.Errorf("times must contain at least one HH:MM value")
	}
	seen := map[string]bool{}
	norm := make([]string, 0, len(cfg.Times))
	for _, v := range cfg.Times {
		v = strings.TrimSpace(unquote(v))
		h, m, err := ParseClock(v)
		if err != nil {
			return Config{}, err
		}
		n := fmt.Sprintf("%02d:%02d", h, m)
		if !seen[n] {
			seen[n] = true
			norm = append(norm, n)
		}
	}
	cfg.Times = norm
	outAcc := make([]string, 0, len(cfg.Accounts))
	for _, a := range cfg.Accounts {
		a = strings.TrimSpace(unquote(a))
		if a != "" {
			outAcc = append(outAcc, a)
		}
	}
	cfg.Accounts = outAcc
	allow := map[string]bool{}
	for _, a := range cfg.Accounts {
		allow[a] = true
	}
	if len(cfg.AccountTimes) > 0 {
		outAT := make(map[string][]string, len(cfg.AccountTimes))
		for acc, list := range cfg.AccountTimes {
			acc = strings.TrimSpace(unquote(acc))
			if acc == "" || !allow[acc] {
				continue
			}
			if len(list) == 0 {
				continue // empty ⇒ inherit (drop key)
			}
			seenAT := map[string]bool{}
			normAT := make([]string, 0, len(list))
			for _, v := range list {
				v = strings.TrimSpace(unquote(v))
				h, m, err := ParseClock(v)
				if err != nil {
					return Config{}, err
				}
				n := fmt.Sprintf("%02d:%02d", h, m)
				if !seenAT[n] {
					seenAT[n] = true
					normAT = append(normAT, n)
				}
			}
			if len(normAT) == 0 {
				continue
			}
			outAT[acc] = normAT
		}
		if len(outAT) == 0 {
			cfg.AccountTimes = nil
		} else {
			cfg.AccountTimes = outAT
		}
	} else {
		cfg.AccountTimes = nil
	}
	cfg.DataDir = strings.TrimSpace(cfg.DataDir)
	cfg.StatePath = strings.TrimSpace(cfg.StatePath)
	cfg.Model = strings.TrimSpace(cfg.Model)
	if cfg.HistoryLimit <= 0 {
		cfg.HistoryLimit = 60
	}
	if cfg.RetryCount < 0 {
		cfg.RetryCount = 0
	}
	return cfg, nil
}


// EffectiveModel returns cfg.Model when non-empty after trim; otherwise fallback.
// Call sites should pass pinger.ModelName as fallback to avoid config→pinger imports.
func EffectiveModel(cfg Config, fallback string) string {
	if m := strings.TrimSpace(cfg.Model); m != "" {
		return m
	}
	return fallback
}

// EffectiveTimes returns the account's custom times when non-empty, otherwise global Times.
func EffectiveTimes(cfg Config, account string) []string {
	if cfg.AccountTimes != nil {
		if custom, ok := cfg.AccountTimes[account]; ok && len(custom) > 0 {
			return append([]string(nil), custom...)
		}
	}
	return append([]string(nil), cfg.Times...)
}

// UnionTimes returns unique sorted HH:MM across allowlisted accounts' effective times.
// If the allowlist is empty, returns a copy of global Times for next-run display stability.
func UnionTimes(cfg Config) []string {
	if len(cfg.Accounts) == 0 {
		return append([]string(nil), cfg.Times...)
	}
	seen := map[string]bool{}
	for _, acc := range cfg.Accounts {
		for _, t := range EffectiveTimes(cfg, acc) {
			seen[t] = true
		}
	}
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// AccountsForSlot returns allowlisted accounts whose effective times contain normalized hhmm.
func AccountsForSlot(cfg Config, hhmm string) []string {
	hhmm = strings.TrimSpace(unquote(hhmm))
	if h, m, err := ParseClock(hhmm); err == nil {
		hhmm = fmt.Sprintf("%02d:%02d", h, m)
	}
	out := make([]string, 0)
	for _, acc := range cfg.Accounts {
		for _, t := range EffectiveTimes(cfg, acc) {
			if t == hhmm {
				out = append(out, acc)
				break
			}
		}
	}
	return out
}

func ParseClock(value string) (int, int, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid time %q: expected HH:MM", value)
	}
	h, e1 := strconv.Atoi(parts[0])
	m, e2 := strconv.Atoi(parts[1])
	if e1 != nil || e2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, fmt.Errorf("invalid time %q: expected 00:00-23:59", value)
	}
	return h, m, nil
}

func parseInlineList(value string) []string {
	value = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(value), "["), "]"))
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, unquote(strings.TrimSpace(p)))
	}
	return out
}

func unquote(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}
