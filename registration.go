package main

import (
	"strings"

	"cpa-plugin-codex-selective-ping/internal/config"
)

const (
	pluginName = "codex-selective-ping"
	version    = "0.1.11"
)

func configFields(cfg config.Config) []map[string]any {
	return []map[string]any{
		{"Name": "schedule_enabled", "Type": "bool", "Description": "Enable daily schedule (not host plugin lifecycle)", "DefaultValue": cfg.Enabled},
		{"Name": "timezone", "Type": "string", "Description": "IANA timezone", "DefaultValue": cfg.Timezone},
		{"Name": "times", "Type": "string", "Description": "Daily HH:MM times", "DefaultValue": strings.Join(cfg.Times, ",")},
		{"Name": "accounts", "Type": "string", "Description": "Whitelist of email/auth_index/name; empty=ping nobody", "DefaultValue": ""},
		{"Name": "data_dir", "Type": "string", "Description": "Optional directory for run_history.json (default: {CPA root}/data/codex-selective-ping)", "DefaultValue": ""},
		{"Name": "state_path", "Type": "string", "Description": "Optional full path to run_history.json (overrides data_dir)", "DefaultValue": ""},
		{"Name": "history_limit", "Type": "int", "Description": "Max persisted run history entries (default 60)", "DefaultValue": cfg.HistoryLimit},
		{"Name": "retry_count", "Type": "int", "Description": "On limited/quota failure, retry this many times after the first attempt, waiting 60s between tries (default 2)", "DefaultValue": cfg.RetryCount},
	}
}

func registrationMeta(cfg config.Config) map[string]any {
	return map[string]any{
		"schema_version": 5,
		"metadata": map[string]any{
			"Name":             pluginName,
			"Version":          version,
			"Author":           "danielhuang-030",
			"GitHubRepository": "https://github.com/danielhuang-030/cpa-plugin-codex-selective-ping",
			"Description":      "Selectively ping configured Codex OAuth accounts on a daily schedule.",
			"ConfigFields":     configFields(cfg),
		},
		"capabilities": map[string]any{"management_api": true},
	}
}
