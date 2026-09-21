package main

import (
	"strings"

	"cpa-plugin-codex-selective-ping/internal/config"
)

const (
	pluginName = "codex-selective-ping"
	version    = "0.1.2"
)

func configFields(cfg config.Config) []map[string]any {
	return []map[string]any{
		{"Name": "schedule_enabled", "Type": "bool", "Description": "Enable daily schedule (not host plugin lifecycle)", "DefaultValue": cfg.Enabled},
		{"Name": "timezone", "Type": "string", "Description": "IANA timezone", "DefaultValue": cfg.Timezone},
		{"Name": "times", "Type": "string", "Description": "Daily HH:MM times", "DefaultValue": strings.Join(cfg.Times, ",")},
		{"Name": "accounts", "Type": "string", "Description": "Whitelist of email/auth_index/name; empty=ping nobody", "DefaultValue": ""},
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
