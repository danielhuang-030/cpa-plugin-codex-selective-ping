package hostapi

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestEnrichQuotaFromListFields(t *testing.T) {
	raw := json.RawMessage(`{
		"plan":"Plus",
		"five_hour":{"remaining":62.5,"resets_at":"2026-09-21T21:40:00+08:00"},
		"weekly":{"used":19,"resets_at":"2026-09-24T08:00:00+08:00"}
	}`)
	got := EnrichQuota(AuthFile{AuthIndex: "1"}, raw, nil)
	if got.Plan != "Plus" {
		t.Fatalf("plan=%q", got.Plan)
	}
	if got.FiveHour == nil || got.FiveHour.Remaining == nil || *got.FiveHour.Remaining != 62.5 {
		t.Fatalf("five_hour=%#v", got.FiveHour)
	}
	if got.FiveHour.ResetsAt == nil {
		t.Fatal("five_hour resets_at missing")
	}
	if got.Weekly == nil || got.Weekly.Used == nil || *got.Weekly.Used != 19 {
		t.Fatalf("weekly=%#v", got.Weekly)
	}
}

func TestEnrichQuotaMissingShowsEmpty(t *testing.T) {
	got := EnrichQuota(AuthFile{AuthIndex: "1", Name: "x"}, json.RawMessage(`{}`), nil)
	if got.Plan != "" || got.FiveHour != nil || got.Weekly != nil {
		t.Fatalf("must not invent: %#v", got)
	}
}

func TestEnrichQuotaRuntimeOverridesWhenPresent(t *testing.T) {
	list := json.RawMessage(`{"plan":"Free"}`)
	runtime := json.RawMessage(`{"plan":"Team","five_hour":{"remaining":10}}`)
	got := EnrichQuota(AuthFile{}, list, runtime)
	if got.Plan != "Team" {
		t.Fatalf("plan=%q", got.Plan)
	}
	if got.FiveHour == nil || got.FiveHour.Remaining == nil || *got.FiveHour.Remaining != 10 {
		t.Fatalf("%#v", got.FiveHour)
	}
}

func TestEnrichQuotaAcceptsAlternateKeys(t *testing.T) {
	raw := json.RawMessage(`{
		"plan_type":"Plus",
		"rate_limit":{"five_hour":{"remaining_fraction":0.5,"reset_at":1690000000},
		"primary_window":{"used_percent":40,"resets_in_seconds":3600}}
	}`)
	// Implementation should accept common aliases when clearly present; if a key is unknown, leave nil.
	got := EnrichQuota(AuthFile{}, raw, nil)
	// plan_type alias
	if got.Plan != "Plus" {
		t.Fatalf("plan alias=%q", got.Plan)
	}
	if got.FiveHour == nil || got.FiveHour.Remaining == nil || *got.FiveHour.Remaining != 0.5 {
		t.Fatalf("five_hour remaining_fraction=%#v", got.FiveHour)
	}
	_ = time.Now()
}

func fakeIDToken(planType string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payloadObj := map[string]any{
		"email": "a@x.com",
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_plan_type":  planType,
			"chatgpt_account_id": "acc-1",
		},
	}
	raw, _ := json.Marshal(payloadObj)
	payload := base64.RawURLEncoding.EncodeToString(raw)
	return header + "." + payload + ".sig"
}

func TestPlanTypeFromIDTokenJWT(t *testing.T) {
	tok := fakeIDToken("plus")
	got := PlanTypeFromIDToken(tok)
	if got != "plus" {
		t.Fatalf("plan_type=%q want plus", got)
	}
	if PlanTypeFromIDToken("not-a-jwt") != "" {
		t.Fatal("invalid jwt must yield empty")
	}
	if PlanTypeFromIDToken("") != "" {
		t.Fatal("empty must yield empty")
	}
}

func TestPlanTypeFromAuthJSONCredential(t *testing.T) {
	tok := fakeIDToken("team")
	raw, _ := json.Marshal(map[string]any{
		"type":         "codex",
		"access_token": "atok",
		"id_token":     tok,
		"email":        "a@x.com",
	})
	got := PlanTypeFromAuthJSON(raw)
	if got != "team" {
		t.Fatalf("got %q want team", got)
	}
}

func TestEnrichQuotaUsesAuthGetWhenListRuntimeLackPlan(t *testing.T) {
	tok := fakeIDToken("plus")
	cred, _ := json.Marshal(map[string]any{"id_token": tok, "access_token": "x"})
	got := EnrichQuota(AuthFile{AuthIndex: "1"}, json.RawMessage(`{}`), nil, cred)
	if got.Plan != "plus" {
		t.Fatalf("plan from AuthGet id_token=%q", got.Plan)
	}
}

func TestEnrichQuotaFromAuthFilesEntryShapes(t *testing.T) {
	// CPA management auth-files exposes decoded id_token.plan_type (not raw JWT).
	entry := json.RawMessage(`{
		"auth_index":"7",
		"name":"alice.json",
		"id_token":{"plan_type":"plus","chatgpt_account_id":"acc"},
		"quota":{"signals":{"note":"observed"}},
		"model_quotas":{}
	}`)
	got := EnrichQuota(AuthFile{AuthIndex: "7"}, entry)
	if got.Plan != "plus" {
		t.Fatalf("plan from id_token.plan_type=%q", got.Plan)
	}
	// Observation signals alone must NOT invent 5h/weekly remaining numbers.
	if got.FiveHour != nil || got.Weekly != nil {
		t.Fatalf("must not invent windows from signals: five=%#v weekly=%#v", got.FiveHour, got.Weekly)
	}
}

func TestQuotaWindowsFromWhamUsage(t *testing.T) {
	raw := json.RawMessage(`{
		"plan_type":"plus",
		"rate_limit":{
			"primary_window":{"used_percent":37.5,"reset_at":1893456000,"limit_window_seconds":18000},
			"secondary_window":{"used_percent":12,"reset_at":1894051200,"limit_window_seconds":604800}
		}
	}`)
	plan, five, weekly := QuotaFromWhamUsage(raw)
	if plan != "plus" {
		t.Fatalf("plan=%q", plan)
	}
	if five == nil || five.Used == nil || *five.Used != 37.5 {
		t.Fatalf("five=%#v", five)
	}
	if five.Remaining == nil || *five.Remaining != 62.5 {
		t.Fatalf("five remaining=%#v", five.Remaining)
	}
	if weekly == nil || weekly.Used == nil || *weekly.Used != 12 {
		t.Fatalf("weekly=%#v", weekly)
	}
	if weekly.Remaining == nil || *weekly.Remaining != 88 {
		t.Fatalf("weekly remaining=%#v", weekly.Remaining)
	}
}
