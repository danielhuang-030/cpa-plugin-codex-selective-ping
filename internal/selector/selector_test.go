package selector

import (
	"testing"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
)

func sample() []hostapi.AuthFile {
	return []hostapi.AuthFile{
		{AuthIndex: "1", Name: "Alice", Email: "alice@Example.com", Provider: "codex"},
		{AuthIndex: "2", Name: "Bob-Work", Email: "bob@x.com", Provider: "codex"},
		{AuthIndex: "3", Name: "gemini", Email: "g@x.com", Provider: "gemini"},
		{AuthIndex: "auth-9", Name: "Spare Codex", Email: "spare@x.com", Type: "codex"},
	}
}

func TestSelectEmptyAccountsReturnsNone(t *testing.T) {
	got := Select(sample(), nil)
	if len(got) != 0 {
		t.Fatalf("got %d want 0", len(got))
	}
	got = Select(sample(), []string{})
	if len(got) != 0 {
		t.Fatalf("got %d want 0", len(got))
	}
}

func TestSelectByEmailCaseInsensitive(t *testing.T) {
	got := Select(sample(), []string{"ALICE@example.com"})
	if len(got) != 1 || got[0].AuthIndex != "1" {
		t.Fatalf("%#v", got)
	}
}

func TestSelectByAuthIndex(t *testing.T) {
	got := Select(sample(), []string{"auth-9"})
	if len(got) != 1 || got[0].AuthIndex != "auth-9" {
		t.Fatalf("%#v", got)
	}
}

func TestSelectByNameCaseInsensitive(t *testing.T) {
	got := Select(sample(), []string{"bob-work"})
	if len(got) != 1 || got[0].AuthIndex != "2" {
		t.Fatalf("%#v", got)
	}
}

func TestSelectExcludesNonCodex(t *testing.T) {
	got := Select(sample(), []string{"gemini", "g@x.com", "3"})
	if len(got) != 0 {
		t.Fatalf("non-codex must be excluded: %#v", got)
	}
}

func TestSelectMultipleStableOrder(t *testing.T) {
	got := Select(sample(), []string{"auth-9", "ALICE@example.com"})
	if len(got) != 2 {
		t.Fatalf("%#v", got)
	}
	// Preserve discovery order from files, not whitelist order.
	if got[0].AuthIndex != "1" || got[1].AuthIndex != "auth-9" {
		t.Fatalf("order=%#v", got)
	}
}

func TestIsCodexIgnoresNameContains(t *testing.T) {
	a := hostapi.AuthFile{AuthIndex: "9", Name: "my-codex-notes", Email: "n@x.com", Provider: "gemini"}
	if IsCodex(a) {
		t.Fatalf("name containing codex must not match when provider/type are non-codex: %#v", a)
	}
	got := Select([]hostapi.AuthFile{a}, []string{"my-codex-notes", "n@x.com", "9"})
	if len(got) != 0 {
		t.Fatalf("must not select gemini account with codex in name: %#v", got)
	}
}
