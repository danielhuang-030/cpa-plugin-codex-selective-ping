package hostapi

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

// PlanTypeFromIDToken extracts chatgpt_plan_type from a Codex id_token JWT payload
// (no signature verification — same approach CPA management uses).
func PlanTypeFromIDToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Some encoders include padding; try StdEncoding with RawURL fallback already failed.
		padded := parts[1]
		switch len(padded) % 4 {
		case 2:
			padded += "=="
		case 3:
			padded += "="
		}
		payload, err = base64.URLEncoding.DecodeString(padded)
		if err != nil {
			return ""
		}
	}
	var claims map[string]json.RawMessage
	if json.Unmarshal(payload, &claims) != nil {
		return ""
	}
	if p := firstString(claims, "chatgpt_plan_type", "plan_type", "planType"); p != "" {
		return p
	}
	for _, key := range []string{"https://api.openai.com/auth", "https://api.openai.com/auth"} {
		raw, ok := claims[key]
		if !ok {
			continue
		}
		var nested map[string]json.RawMessage
		if json.Unmarshal(raw, &nested) != nil {
			continue
		}
		if p := firstString(nested, "chatgpt_plan_type", "chatgpt_plan_type", "plan_type", "planType"); p != "" {
			return p
		}
	}
	return ""
}

// PlanTypeFromAuthJSON reads credential JSON from AuthGet and returns plan from id_token.
func PlanTypeFromAuthJSON(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	if p := firstString(m, "plan", "plan_type", "planType"); p != "" {
		return p
	}
	rawTok, ok := m["id_token"]
	if !ok {
		return ""
	}
	var tok string
	if json.Unmarshal(rawTok, &tok) == nil {
		return PlanTypeFromIDToken(tok)
	}
	// Already-decoded claims object (auth-files shape).
	var nested map[string]json.RawMessage
	if json.Unmarshal(rawTok, &nested) == nil {
		if p := firstString(nested, "plan_type", "planType", "chatgpt_plan_type"); p != "" {
			return p
		}
	}
	return ""
}

// QuotaFromWhamUsage maps ChatGPT backend-api/wham/usage JSON into plan + 5h/weekly.
// remaining = 100 - used_percent when used_percent is present. Does not invent numbers.
func QuotaFromWhamUsage(raw json.RawMessage) (plan string, fiveHour, weekly *QuotaWindow) {
	if len(raw) == 0 {
		return "", nil, nil
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(raw, &root) != nil {
		return "", nil, nil
	}
	plan = firstString(root, "plan_type", "planType", "plan")
	rateRaw, ok := root["rate_limit"]
	if !ok {
		rateRaw = root["rateLimit"]
	}
	if len(rateRaw) == 0 {
		return plan, nil, nil
	}
	var rate map[string]json.RawMessage
	if json.Unmarshal(rateRaw, &rate) != nil {
		return plan, nil, nil
	}
	primary := firstRaw(rate, "primary_window", "primaryWindow")
	secondary := firstRaw(rate, "secondary_window", "secondaryWindow")
	// Prefer windows matched by limit duration when both present under either key.
	fiveHour = windowFromWham(primary)
	weekly = windowFromWham(secondary)
	// Also try matching by limit_window_seconds across both.
	if fiveHour == nil || weekly == nil {
		for _, cand := range []json.RawMessage{primary, secondary} {
			if len(cand) == 0 {
				continue
			}
			secs := windowSeconds(cand)
			if fiveHour == nil && secs == 18000 {
				fiveHour = windowFromWham(cand)
			}
			if weekly == nil && (secs == 604800 || (secs >= 2419200 && secs <= 2678400)) {
				weekly = windowFromWham(cand)
			}
		}
	}
	return plan, fiveHour, weekly
}

func firstRaw(m map[string]json.RawMessage, keys ...string) json.RawMessage {
	for _, k := range keys {
		if raw, ok := m[k]; ok {
			return raw
		}
	}
	return nil
}

func windowSeconds(raw json.RawMessage) float64 {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return 0
	}
	if v, ok := asFloat(m, "limit_window_seconds", "limitWindowSeconds"); ok {
		return v
	}
	return 0
}

func windowFromWham(raw json.RawMessage) *QuotaWindow {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	w := &QuotaWindow{}
	found := false
	if used, ok := asFloat(m, "used_percent", "usedPercent", "used_fraction", "usedFraction"); ok {
		w.Used = &used
		rem := 100 - used
		w.Remaining = &rem
		found = true
	} else if rem, ok := asFloat(m, "remaining_percent", "remainingPercent", "remaining_fraction", "remainingFraction", "remaining"); ok {
		w.Remaining = &rem
		found = true
	}
	if t, ok := asTime(m, "reset_at", "resetAt", "resets_at", "resetsAt"); ok {
		w.ResetsAt = &t
		found = true
	} else if sec, ok := asFloat(m, "reset_after_seconds", "resetAfterSeconds", "resets_in_seconds", "resetsInSeconds"); ok {
		t := time.Now().Add(time.Duration(sec) * time.Second)
		w.ResetsAt = &t
		found = true
	}
	if !found {
		return nil
	}
	return w
}
