package runstate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	pluginDataSubdir     = "codex-selective-ping"
	historyFileName      = "run_history.json"
	legacyLastRunFileName = "last_run.json"
)

type historyFile struct {
	Version int       `json:"version"`
	Runs    []Summary `json:"runs"`
}

// NewPersisted returns a State that loads/saves run history at path (best-effort).
func NewPersisted(path string) *State {
	s := New()
	s.SetPersistPath(path)
	return s
}

// SetPersistPath configures disk persistence and best-effort loads history.
// Load failures leave history empty and never panic.
func (s *State) SetPersistPath(path string) {
	s.mu.Lock()
	s.persistPath = path
	s.mu.Unlock()
	s.loadHistory()
}

// SetHistoryLimit sets the max number of persisted runs (≤0 → 60).
func (s *State) SetHistoryLimit(n int) {
	if n <= 0 {
		n = 60
	}
	s.mu.Lock()
	s.historyLimit = n
	s.mu.Unlock()
}

func (s *State) loadHistory() {
	s.mu.RLock()
	path := s.persistPath
	s.mu.RUnlock()
	if path == "" || isForbiddenPersistPath(path) {
		return
	}
	if runs, ok := readHistoryFile(path); ok {
		s.applyRuns(runs)
		return
	}
	legacy := filepath.Join(filepath.Dir(path), legacyLastRunFileName)
	if filepath.Base(path) == legacyLastRunFileName {
		legacy = path
	}
	if sum, ok := readLegacyLastRun(legacy); ok {
		s.applyRuns([]Summary{sum})
		s.persistHistory()
		if legacy != path {
			_ = os.Remove(legacy)
		}
	}
}

func readHistoryFile(path string) ([]Summary, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var hf historyFile
	if err := json.Unmarshal(data, &hf); err == nil && hf.Version >= 1 {
		return cloneRuns(hf.Runs), true
	}
	// Accidentally pointed state_path at a legacy single Summary file.
	var sum Summary
	if err := json.Unmarshal(data, &sum); err == nil && !sum.At.IsZero() {
		return []Summary{sum}, true
	}
	return nil, false
}

func readLegacyLastRun(path string) (Summary, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Summary{}, false
	}
	var sum Summary
	if err := json.Unmarshal(data, &sum); err != nil {
		return Summary{}, false
	}
	return sum, true
}

func (s *State) applyRuns(runs []Summary) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = cloneRuns(runs)
	if len(s.history) > 0 {
		cp := s.history[0]
		cp.Accounts = append([]AccountResult(nil), s.history[0].Accounts...)
		s.lastRun = &cp
	} else {
		s.lastRun = nil
	}
}

func (s *State) persistHistory() {
	s.mu.RLock()
	path := s.persistPath
	limit := s.historyLimit
	runs := cloneRuns(s.history)
	s.mu.RUnlock()
	if path == "" || isForbiddenPersistPath(path) {
		return
	}
	if limit <= 0 {
		limit = 60
	}
	if len(runs) > limit {
		runs = runs[:limit]
	}
	data, err := json.MarshalIndent(historyFile{Version: 1, Runs: runs}, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		_ = os.WriteFile(path, data, 0o644)
	}
}

func cloneRuns(in []Summary) []Summary {
	if in == nil {
		return nil
	}
	out := make([]Summary, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].Accounts = append([]AccountResult(nil), in[i].Accounts...)
	}
	return out
}

// DefaultStatePath is {CPA root}/data/codex-selective-ping/run_history.json
// where CPA root is the parent of a discovered plugins/ directory (or empty).
func DefaultStatePath() string {
	root := resolveCPARoot()
	if root == "" {
		return ""
	}
	return filepath.Join(root, "data", pluginDataSubdir, historyFileName)
}

// ResolveStatePath applies optional state_path / data_dir overrides.
// Relative paths are resolved against process cwd. Empty overrides use DefaultStatePath.
func ResolveStatePath(statePath, dataDir string) string {
	if sp := strings.TrimSpace(statePath); sp != "" {
		return absAgainstCwd(sp)
	}
	if dd := strings.TrimSpace(dataDir); dd != "" {
		return filepath.Join(absAgainstCwd(dd), historyFileName)
	}
	return DefaultStatePath()
}

func absAgainstCwd(p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(cwd, p))
}

func resolveCPARoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := cwd
	for i := 0; i < 8; i++ {
		if st, err := os.Stat(filepath.Join(dir, "plugins")); err == nil && st.IsDir() {
			return dir
		}
		if filepath.Base(dir) == "plugins" {
			return filepath.Dir(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func isForbiddenPersistPath(path string) bool {
	if path == "" {
		return false
	}
	slash := filepath.ToSlash(filepath.Clean(path))
	return strings.Contains(slash, "/auths/") || strings.HasPrefix(slash, "auths/")
}
