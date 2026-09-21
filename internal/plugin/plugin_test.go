package plugin

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"cpa-plugin-codex-selective-ping/internal/config"
	"cpa-plugin-codex-selective-ping/internal/hostapi"
)

type mockHost struct {
	mu      sync.Mutex
	block   chan struct{}
	listed  bool
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
		Enabled:  false,
		Timezone: "UTC",
		Times:    []string{"21:00"},
		Accounts: []string{"a@x.com"},
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
