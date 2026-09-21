package runstate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPersistLastRunRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "last_run.json")
	s1 := NewPersisted(path)
	if !s1.TryBegin() {
		t.Fatal("begin")
	}
	at := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	sum := Summary{
		At:        at,
		Mode:      "manual",
		Total:     2,
		Attempted: 1,
		Succeeded: 1,
		Failed:    0,
		Limited:   0,
		Skipped:   1,
		Message:   "done",
		Accounts: []AccountResult{
			{AuthIndex: "1", Name: "alice", Email: "a@x.com", Status: "success", Attempts: 1},
			{AuthIndex: "2", Name: "bob", Status: "skipped"},
		},
	}
	s1.End(sum)

	s2 := NewPersisted(path)
	snap := s2.Snapshot(nil, nil, time.Time{})
	if snap.LastRun == nil {
		t.Fatal("expected LastRun after reload")
	}
	got := snap.LastRun
	if !got.At.Equal(at) || got.Mode != "manual" || got.Total != 2 || got.Attempted != 1 || got.Succeeded != 1 || got.Skipped != 1 {
		t.Fatalf("summary mismatch: %#v", got)
	}
	if got.Message != "done" {
		t.Fatalf("message=%q", got.Message)
	}
	if len(got.Accounts) != 2 {
		t.Fatalf("accounts=%d", len(got.Accounts))
	}
	if got.Accounts[0].AuthIndex != "1" || got.Accounts[0].Status != "success" || got.Accounts[0].Email != "a@x.com" {
		t.Fatalf("account0=%#v", got.Accounts[0])
	}
	if got.Accounts[1].AuthIndex != "2" || got.Accounts[1].Status != "skipped" {
		t.Fatalf("account1=%#v", got.Accounts[1])
	}
}

func TestPersistMissingFileYieldsEmptyLastRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "last_run.json")
	s := NewPersisted(path)
	snap := s.Snapshot(nil, nil, time.Time{})
	if snap.LastRun != nil {
		t.Fatalf("expected nil LastRun, got %#v", snap.LastRun)
	}
}

func TestPersistCorruptJSONYieldsEmptyNoPanic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "last_run.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewPersisted(path)
	snap := s.Snapshot(nil, nil, time.Time{})
	if snap.LastRun != nil {
		t.Fatalf("corrupt file must yield empty LastRun, got %#v", snap.LastRun)
	}
	if !s.TryBegin() {
		t.Fatal("begin")
	}
	s.End(Summary{At: time.Now().UTC(), Mode: "manual", Message: "recovered"})
	s3 := NewPersisted(path)
	if s3.Snapshot(nil, nil, time.Time{}).LastRun == nil {
		t.Fatal("expected LastRun after overwrite")
	}
}

func TestResolveStatePathOverrides(t *testing.T) {
	absState := filepath.Join(t.TempDir(), "custom.json")
	if got := ResolveStatePath(absState, ""); got != absState {
		t.Fatalf("state_path override: got %q want %q", got, absState)
	}
	dataDir := t.TempDir()
	want := filepath.Join(dataDir, "run_history.json")
	if got := ResolveStatePath("", dataDir); got != want {
		t.Fatalf("data_dir override: got %q want %q", got, want)
	}
	if got := ResolveStatePath(absState, dataDir); got != absState {
		t.Fatalf("state_path must win: got %q", got)
	}
}

func TestDefaultStatePathUnderDataNotAuths(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "plugins"), 0o755); err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	got := DefaultStatePath()
	want := filepath.Join(root, "data", "codex-selective-ping", "run_history.json")
	if got != want {
		t.Fatalf("DefaultStatePath=%q want %q", got, want)
	}
	slash := filepath.ToSlash(got)
	if strings.Contains(slash, "/auths/") {
		t.Fatalf("must not write under auths: %q", got)
	}
}

func TestResolveRelativePathsUseCwd(t *testing.T) {
	root := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	got := ResolveStatePath("rel-state.json", "")
	want := filepath.Join(root, "rel-state.json")
	if got != want {
		t.Fatalf("relative state_path: got %q want %q", got, want)
	}
	got = ResolveStatePath("", "mydata")
	want = filepath.Join(root, "mydata", "run_history.json")
	if got != want {
		t.Fatalf("relative data_dir: got %q want %q", got, want)
	}
}

func TestPersistSkipsForbiddenAuthsPath(t *testing.T) {
	root := t.TempDir()
	forbidden := filepath.Join(root, "auths", "plugins", "codex-selective-ping", "last_run.json")
	s := NewPersisted(forbidden)
	if !s.TryBegin() {
		t.Fatal("begin")
	}
	s.End(Summary{At: time.Now().UTC(), Mode: "manual", Message: "should not land in auths"})
	if _, err := os.Stat(forbidden); err == nil {
		t.Fatalf("must not write under auths/plugins: %s", forbidden)
	}
	if s.Snapshot(nil, nil, time.Time{}).LastRun == nil {
		t.Fatal("in-memory LastRun must remain even when persist skipped")
	}
}


func TestDefaultStatePathWithoutPluginsIsEmpty(t *testing.T) {
	root := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	if got := DefaultStatePath(); got != "" {
		t.Fatalf("expected empty default without plugins/, got %q", got)
	}
}

func TestPersistHistoryRoundTripNewestFirst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "run_history.json")
	s1 := NewPersisted(path)
	s1.SetHistoryLimit(60)
	if !s1.TryBegin() {
		t.Fatal("begin1")
	}
	s1.End(Summary{At: time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC), Mode: "schedule", Message: "first"})
	if !s1.TryBegin() {
		t.Fatal("begin2")
	}
	s1.End(Summary{At: time.Date(2026, 9, 21, 11, 0, 0, 0, time.UTC), Mode: "manual", Message: "second"})

	s2 := NewPersisted(path)
	snap := s2.Snapshot(nil, nil, time.Time{})
	if snap.LastRun == nil || snap.LastRun.Message != "second" {
		t.Fatalf("LastRun=%#v", snap.LastRun)
	}
	if len(snap.RunHistory) != 2 {
		t.Fatalf("RunHistory len=%d want 2: %#v", len(snap.RunHistory), snap.RunHistory)
	}
	if snap.RunHistory[0].Message != "second" || snap.RunHistory[1].Message != "first" {
		t.Fatalf("order=%#v", snap.RunHistory)
	}
}

func TestPersistHistoryTrimsToLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "run_history.json")
	s := NewPersisted(path)
	s.SetHistoryLimit(2)
	for i := 0; i < 5; i++ {
		if !s.TryBegin() {
			t.Fatalf("begin %d", i)
		}
		s.End(Summary{At: time.Date(2026, 9, 21, i, 0, 0, 0, time.UTC), Mode: "manual", Message: string(rune('a' + i))})
	}
	s2 := NewPersisted(path)
	s2.SetHistoryLimit(2)
	snap := s2.Snapshot(nil, nil, time.Time{})
	if len(snap.RunHistory) != 2 {
		t.Fatalf("len=%d want 2", len(snap.RunHistory))
	}
	if snap.RunHistory[0].Message != "e" || snap.RunHistory[1].Message != "d" {
		t.Fatalf("kept=%#v", snap.RunHistory)
	}
}

func TestMigrateLegacyLastRunJSON(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "last_run.json")
	history := filepath.Join(dir, "run_history.json")
	sum := Summary{At: time.Date(2026, 9, 20, 8, 0, 0, 0, time.UTC), Mode: "manual", Message: "legacy", Succeeded: 1}
	data, err := jsonMarshalForTest(sum)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, data, 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewPersisted(history)
	snap := s.Snapshot(nil, nil, time.Time{})
	if snap.LastRun == nil || snap.LastRun.Message != "legacy" {
		t.Fatalf("LastRun=%#v", snap.LastRun)
	}
	if len(snap.RunHistory) != 1 {
		t.Fatalf("history=%#v", snap.RunHistory)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy must be deleted, err=%v", err)
	}
	if _, err := os.Stat(history); err != nil {
		t.Fatalf("history file missing: %v", err)
	}
}

func TestDefaultStatePathUsesRunHistoryFilename(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "plugins"), 0o755); err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	got := DefaultStatePath()
	want := filepath.Join(root, "data", "codex-selective-ping", "run_history.json")
	if got != want {
		t.Fatalf("DefaultStatePath=%q want %q", got, want)
	}
}

func TestResolveDataDirUsesRunHistoryFilename(t *testing.T) {
	dataDir := t.TempDir()
	want := filepath.Join(dataDir, "run_history.json")
	if got := ResolveStatePath("", dataDir); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func jsonMarshalForTest(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
