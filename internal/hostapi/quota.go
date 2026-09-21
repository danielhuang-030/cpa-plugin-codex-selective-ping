package hostapi

import (
	"encoding/json"
	"strings"
	"time"
)

func EnrichQuota(a AuthFile, blobs ...json.RawMessage) AuthFile {
	for _, blob := range blobs {
		if len(blob) == 0 {
			continue
		}
		var m map[string]json.RawMessage
		if json.Unmarshal(blob, &m) != nil {
			continue
		}
		if p := firstString(m, "plan", "plan_type", "planType"); p != "" {
			a.Plan = p
		}
		// Nested id_token: either raw JWT string or decoded claims (auth-files).
		if raw, ok := m["id_token"]; ok {
			if p := PlanTypeFromAuthJSON([]byte(`{"id_token":` + string(raw) + `}`)); p != "" {
				a.Plan = p
			}
		}
		// Whole credential JSON (AuthGet) may carry id_token at top level.
		if a.Plan == "" {
			if p := PlanTypeFromAuthJSON(blob); p != "" {
				a.Plan = p
			}
		}
		if w := parseWindow(m, "five_hour", "fiveHour", "rate_limit_five_hour"); w != nil {
			a.FiveHour = w
		}
		if w := parseWindow(m, "weekly", "week", "weekly_limit"); w != nil {
			a.Weekly = w
		}
		// Nested rate_limit / primary_window best-effort (only if fields exist).
		if raw, ok := m["rate_limit"]; ok {
			var nested map[string]json.RawMessage
			if json.Unmarshal(raw, &nested) == nil {
				if w := parseWindow(nested, "five_hour", "fiveHour"); w != nil {
					a.FiveHour = w
				}
				if w := parseWindow(nested, "weekly", "week"); w != nil {
					a.Weekly = w
				}
			}
		}
		if raw, ok := m["primary_window"]; ok {
			if w := parseWindowMap(raw); w != nil && a.FiveHour == nil {
				a.FiveHour = w
			}
		}
	}
	return a
}

func parseWindow(m map[string]json.RawMessage, keys ...string) *QuotaWindow {
	for _, k := range keys {
		if raw, ok := m[k]; ok {
			if w := parseWindowMap(raw); w != nil {
				return w
			}
		}
	}
	return nil
}

func parseWindowMap(raw json.RawMessage) *QuotaWindow {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	w := &QuotaWindow{}
	found := false
	if v, ok := asFloat(m, "remaining", "remaining_fraction", "remainingFraction", "remaining_percent", "remainingPercent"); ok {
		w.Remaining = &v
		found = true
	}
	if v, ok := asFloat(m, "used", "used_percent", "usedPercent", "used_fraction", "usedFraction"); ok {
		w.Used = &v
		found = true
	}
	if t, ok := asTime(m, "resets_at", "reset_at", "resetsAt", "resetAt"); ok {
		w.ResetsAt = &t
		found = true
	} else if sec, ok := asFloat(m, "resets_in_seconds", "resetsInSeconds"); ok {
		t := time.Now().Add(time.Duration(sec) * time.Second)
		w.ResetsAt = &t
		found = true
	}
	if !found {
		return nil
	}
	return w
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

func asFloat(m map[string]json.RawMessage, keys ...string) (float64, bool) {
	for _, k := range keys {
		raw, ok := m[k]
		if !ok {
			continue
		}
		var f float64
		if json.Unmarshal(raw, &f) == nil {
			return f, true
		}
		var s string
		if json.Unmarshal(raw, &s) == nil {
			var f2 float64
			if json.Unmarshal([]byte(s), &f2) == nil {
				return f2, true
			}
		}
	}
	return 0, false
}

func asTime(m map[string]json.RawMessage, keys ...string) (time.Time, bool) {
	for _, k := range keys {
		raw, ok := m[k]
		if !ok {
			continue
		}
		var s string
		if json.Unmarshal(raw, &s) == nil && strings.TrimSpace(s) != "" {
			if t, err := time.Parse(time.RFC3339, strings.TrimSpace(s)); err == nil {
				return t, true
			}
		}
		var unix int64
		if json.Unmarshal(raw, &unix) == nil && unix > 0 {
			return time.Unix(unix, 0), true
		}
		var f float64
		if json.Unmarshal(raw, &f) == nil && f > 0 {
			return time.Unix(int64(f), 0), true
		}
	}
	return time.Time{}, false
}
