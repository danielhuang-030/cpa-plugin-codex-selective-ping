package selector

import (
	"strings"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
)

func IsCodex(a hostapi.AuthFile) bool {
	p := strings.ToLower(strings.TrimSpace(a.Provider))
	typ := strings.ToLower(strings.TrimSpace(a.Type))
	return p == "codex" || typ == "codex" ||
		strings.Contains(p, "codex") || strings.Contains(typ, "codex")
}

func Select(files []hostapi.AuthFile, accounts []string) []hostapi.AuthFile {
	if len(accounts) == 0 {
		return nil
	}
	norms := make([]string, 0, len(accounts))
	for _, a := range accounts {
		a = strings.TrimSpace(a)
		if a != "" {
			norms = append(norms, a)
		}
	}
	if len(norms) == 0 {
		return nil
	}
	out := make([]hostapi.AuthFile, 0)
	for _, f := range files {
		if !IsCodex(f) {
			continue
		}
		if matches(f, norms) {
			out = append(out, f)
		}
	}
	return out
}

func matches(f hostapi.AuthFile, accounts []string) bool {
	idx := strings.TrimSpace(f.AuthIndex)
	email := strings.ToLower(strings.TrimSpace(f.Email))
	name := strings.ToLower(strings.TrimSpace(f.Name))
	account := strings.ToLower(strings.TrimSpace(f.Account))
	for _, raw := range accounts {
		if idx != "" && raw == idx {
			return true
		}
		want := strings.ToLower(raw)
		if email != "" && want == email {
			return true
		}
		if name != "" && want == name {
			return true
		}
		if account != "" && want == account {
			return true
		}
	}
	return false
}
