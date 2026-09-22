package plugin

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"cpa-plugin-codex-selective-ping/internal/config"
	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/runstate"
)

type mockHost struct {
	mu     sync.Mutex
	block  chan struct{}
	listed bool
}

func (m *mockHost) AuthList(ctx context.Context) ([]hostapi.AuthFile, error) {
	m.mu.Lock()
	m.listed = true
	m.mu.Unlock()
	if m.block != nil {
		select {
		case <-m.block:
		case <-ctx.Done():
		}
	}
	return []hostapi.AuthFile{{AuthIndex: "1", Name: "a", Email: "a@x.com", Provider: "codex"}}, nil
}
func (m *mockHost) AuthGet(context.Context, string) ([]byte, error) {
	return []byte(`{"access_token":"tok"}`), nil
}
func (m *mockHost) HTTPDo(ctx context.Context, req hostapi.HTTPRequest) (hostapi.HTTPResponse, error) {
	if m.block != nil {
		select {
		case <-m.block:
		case <-ctx.Done():
		}
	}
	return hostapi.HTTPResponse{StatusCode: 200}, nil
}
func (m *mockHost) AuthGetRuntime(context.Context, string) (json.RawMessage, error) {
	return nil, hostapi.ErrUnsupported
}

func TestApplyConfigStoresConfig(t *testing.T) {
	h := &mockHost{}
	p := New(h, "0.1.0")
	defer p.Shutdown()
	cfg := config.Config{
		Enabled:  false,
		Timezone: "UTC",
		Times:    []string{"09:30", "18:00"},
		Accounts: []string{"a@x.com", "auth-1"},
	}
	p.ApplyConfig(cfg)
	got := p.Config()
	if got.Enabled != false {
		t.Fatalf("Enabled=%v", got.Enabled)
	}
	if got.Timezone != "UTC" {
		t.Fatalf("Timezone=%q", got.Timezone)
	}
	if len(got.Times) != 2 || got.Times[0] != "09:30" || got.Times[1] != "18:00" {
		t.Fatalf("Times=%v", got.Times)
	}
	if len(got.Accounts) != 2 || got.Accounts[0] != "a@x.com" || got.Accounts[1] != "auth-1" {
		t.Fatalf("Accounts=%v", got.Accounts)
	}
}

func TestStartManualRunCollision(t *testing.T) {
	block := make(chan struct{})
	h := &mockHost{block: block}
	p := New(h, "0.1.0")
	defer func() {
		close(block)
		p.Shutdown()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if !p.State.Snapshot(nil, nil, time.Time{}).Running {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	p.ApplyConfig(config.Config{
		Enabled:   false,
		Timezone:  "UTC",
		Times:     []string{"21:00"},
		Accounts:  []string{"a@x.com"},
		StatePath: filepath.Join(t.TempDir(), "last_run.json"),
	})
	if !p.StartManualRun() {
		t.Fatal("first StartManualRun should return true")
	}
	// Immediately colliding while claimed/running
	if p.StartManualRun() {
		t.Fatal("second StartManualRun while running should return false")
	}
	if p.State.TryBegin() {
		t.Fatal("TryBegin while running should fail")
	}
}

func TestApplyConfigLoadsPersistedLastRun(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "last_run.json")
	seed := runstate.NewPersisted(statePath)
	if !seed.TryBegin() {
		t.Fatal("begin")
	}
	at := time.Date(2026, 9, 21, 15, 0, 0, 0, time.UTC)
	seed.End(runstate.Summary{At: at, Mode: "scheduled", Succeeded: 1, Message: "seeded"})

	h := &mockHost{}
	p := New(h, "0.1.5")
	defer p.Shutdown()
	p.ApplyConfig(config.Config{
		Enabled:   false,
		Timezone:  "UTC",
		Times:     []string{"21:00"},
		Accounts:  []string{"a@x.com"},
		StatePath: statePath,
	})
	snap := p.State.Snapshot(nil, nil, time.Time{})
	if snap.LastRun == nil || snap.LastRun.Message != "seeded" || snap.LastRun.Mode != "scheduled" {
		t.Fatalf("expected loaded LastRun, got %#v", snap.LastRun)
	}
}

func samplePerAccountCfg() config.Config {
	return config.Config{
		Enabled:  true,
		Timezone: "Asia/Taipei",
		Times:    []string{"06:00", "11:00", "16:00", "21:00"},
		Accounts: []string{"alice@example.com", "bob@example.com"},
		AccountTimes: map[string][]string{
			"bob@example.com": {"07:30", "19:00"},
		},
	}
}

func TestConfigForScheduledFireFiltersBySlot(t *testing.T) {
	base := samplePerAccountCfg()
	loc, err := time.LoadLocation(base.Timezone)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		at   time.Time
		want []string
	}{
		{
			name: "06:00 only inheriting alice",
			at:   time.Date(2026, 9, 22, 6, 0, 0, 0, loc),
			want: []string{"alice@example.com"},
		},
		{
			name: "07:30 only custom bob",
			at:   time.Date(2026, 9, 22, 7, 30, 0, 0, loc),
			want: []string{"bob@example.com"},
		},
		{
			name: "11:00 only inheriting alice",
			at:   time.Date(2026, 9, 22, 11, 0, 0, 0, loc),
			want: []string{"alice@example.com"},
		},
		{
			name: "19:00 only custom bob",
			at:   time.Date(2026, 9, 22, 19, 0, 0, 0, loc),
			want: []string{"bob@example.com"},
		},
		{
			name: "12:00 empty nobody matches",
			at:   time.Date(2026, 9, 22, 12, 0, 0, 0, loc),
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := configForScheduledFire(base, tt.at)
			if !ok {
				t.Fatal("expected ok=true for valid timezone")
			}
			if len(got.Accounts) != len(tt.want) {
				t.Fatalf("Accounts=%v want %v", got.Accounts, tt.want)
			}
			for i := range tt.want {
				if got.Accounts[i] != tt.want[i] {
					t.Fatalf("Accounts=%v want %v", got.Accounts, tt.want)
				}
			}
			// Original allowlist must stay intact (deep copy before filter).
			if len(base.Accounts) != 2 || base.Accounts[0] != "alice@example.com" {
				t.Fatalf("base.Accounts mutated: %v", base.Accounts)
			}
		})
	}
}

func TestConfigForScheduledFireInvalidTimezone(t *testing.T) {
	cfg := samplePerAccountCfg()
	cfg.Timezone = "Not/AZone"
	_, ok := configForScheduledFire(cfg, time.Date(2026, 9, 22, 6, 0, 0, 0, time.UTC))
	if ok {
		t.Fatal("invalid timezone must skip scheduled fire (ok=false)")
	}
}

func TestConfigDeepCopiesAccountTimes(t *testing.T) {
	h := &mockHost{}
	p := New(h, "0.1.0")
	defer p.Shutdown()
	p.ApplyConfig(samplePerAccountCfg())

	got := p.Config()
	if len(got.AccountTimes) != 1 {
		t.Fatalf("AccountTimes=%#v", got.AccountTimes)
	}
	got.AccountTimes["bob@example.com"][0] = "00:00"
	got.AccountTimes["evil@example.com"] = []string{"01:00"}
	got.Accounts[0] = "mutated"
	got.Times[0] = "00:00"

	again := p.Config()
	if again.Accounts[0] != "alice@example.com" {
		t.Fatalf("Accounts mutated via Config() return: %v", again.Accounts)
	}
	if again.Times[0] != "06:00" {
		t.Fatalf("Times mutated via Config() return: %v", again.Times)
	}
	bob := again.AccountTimes["bob@example.com"]
	if len(bob) != 2 || bob[0] != "07:30" {
		t.Fatalf("AccountTimes slice mutated: %v", bob)
	}
	if _, ok := again.AccountTimes["evil@example.com"]; ok {
		t.Fatalf("AccountTimes map mutated: %#v", again.AccountTimes)
	}
}

func TestStartManualRunUsesFullAllowlist(t *testing.T) {
	h := &mockHostFull{}
	p := New(h, "0.1.0")
	defer p.Shutdown()
	cfg := samplePerAccountCfg()
	cfg.Enabled = false // no scheduler loop
	cfg.StatePath = filepath.Join(t.TempDir(), "last_run.json")
	p.ApplyConfig(cfg)

	if !p.StartManualRun() {
		t.Fatal("StartManualRun should succeed")
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		snap := p.State.Snapshot(nil, nil, time.Time{})
		if !snap.Running && snap.LastRun != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	snap := p.State.Snapshot(nil, nil, time.Time{})
	if snap.LastRun == nil {
		t.Fatal("expected LastRun")
	}
	if snap.LastRun.Mode != "force" {
		t.Fatalf("Mode=%q want force", snap.LastRun.Mode)
	}
	emails := map[string]bool{}
	for _, a := range snap.LastRun.Accounts {
		emails[a.Email] = true
	}
	if !emails["alice@example.com"] || !emails["bob@example.com"] {
		t.Fatalf("manual/force must use full allowlist, got accounts=%#v", snap.LastRun.Accounts)
	}
}

// mockHostFull returns both alice and bob so runner can select full allowlist.
type mockHostFull struct{}

func (m *mockHostFull) AuthList(ctx context.Context) ([]hostapi.AuthFile, error) {
	return []hostapi.AuthFile{
		{AuthIndex: "1", Name: "alice", Email: "alice@example.com", Provider: "codex"},
		{AuthIndex: "2", Name: "bob", Email: "bob@example.com", Provider: "codex"},
	}, nil
}
func (m *mockHostFull) AuthGet(context.Context, string) ([]byte, error) {
	return []byte(`{"access_token":"tok"}`), nil
}
func (m *mockHostFull) HTTPDo(ctx context.Context, req hostapi.HTTPRequest) (hostapi.HTTPResponse, error) {
	return hostapi.HTTPResponse{StatusCode: 200}, nil
}
func (m *mockHostFull) AuthGetRuntime(context.Context, string) (json.RawMessage, error) {
	return nil, hostapi.ErrUnsupported
}

func TestApplyConfigClonesOnWrite(t *testing.T) {
	h := &mockHost{}
	p := New(h, "0.1.0")
	defer p.Shutdown()

	cfg := samplePerAccountCfg()
	cfg.Enabled = false
	p.ApplyConfig(cfg)

	cfg.Accounts[0] = "mutated-by-caller"
	cfg.Times[0] = "00:00"
	cfg.AccountTimes["bob@example.com"][0] = "00:00"
	cfg.AccountTimes["evil@example.com"] = []string{"01:00"}

	got := p.Config()
	if got.Accounts[0] != "alice@example.com" {
		t.Fatalf("ApplyConfig must clone Accounts on write; got %v", got.Accounts)
	}
	if got.Times[0] != "06:00" {
		t.Fatalf("ApplyConfig must clone Times on write; got %v", got.Times)
	}
	bob := got.AccountTimes["bob@example.com"]
	if len(bob) != 2 || bob[0] != "07:30" {
		t.Fatalf("ApplyConfig must clone AccountTimes on write; got %v", bob)
	}
	if _, ok := got.AccountTimes["evil@example.com"]; ok {
		t.Fatalf("ApplyConfig must clone AccountTimes map on write; got %#v", got.AccountTimes)
	}
}
