package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var defaultTimes = []string{"06:00", "11:00", "16:00", "21:00"}

type Config struct {
	Enabled   bool     `json:"schedule_enabled"`
	Timezone  string   `json:"timezone"`
	Times     []string `json:"times"`
	Accounts  []string `json:"accounts"`
	DataDir      string `json:"data_dir,omitempty"`
	StatePath    string `json:"state_path,omitempty"`
	HistoryLimit int    `json:"history_limit,omitempty"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:      true,
		Timezone:     "Asia/Taipei",
		Times:        append([]string(nil), defaultTimes...),
		Accounts:     []string{},
		HistoryLimit: 60,
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
			ScheduleEnabled *bool    `json:"schedule_enabled"`
			Timezone        string   `json:"timezone"`
			Times           []string `json:"times"`
			Accounts        []string `json:"accounts"`
			DataDir         string `json:"data_dir"`
			StatePath       string `json:"state_path"`
			HistoryLimit    *int   `json:"history_limit"`
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
		if strings.TrimSpace(p.DataDir) != "" {
			cfg.DataDir = strings.TrimSpace(p.DataDir)
		}
		if strings.TrimSpace(p.StatePath) != "" {
			cfg.StatePath = strings.TrimSpace(p.StatePath)
		}
		if p.HistoryLimit != nil {
			cfg.HistoryLimit = *p.HistoryLimit
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
	for _, rawLine := range lines {
		line := strings.TrimSpace(strings.SplitN(rawLine, "#", 2)[0])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "-") && (mode == "times" || mode == "accounts") {
			item := unquote(strings.TrimSpace(strings.TrimPrefix(line, "-")))
			if mode == "times" {
				times = append(times, item)
			} else {
				accounts = append(accounts, item)
			}
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		mode = ""
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
		}
	}
	if times != nil {
		cfg.Times = times
	}
	if accounts != nil {
		cfg.Accounts = accounts
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
	cfg.DataDir = strings.TrimSpace(cfg.DataDir)
	cfg.StatePath = strings.TrimSpace(cfg.StatePath)
	if cfg.HistoryLimit <= 0 {
		cfg.HistoryLimit = 60
	}
	return cfg, nil
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
