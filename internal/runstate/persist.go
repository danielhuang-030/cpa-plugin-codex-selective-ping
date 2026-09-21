package runstate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	pluginDataSubdir = "codex-selective-ping"
	lastRunFileName  = "last_run.json"
)

// NewPersisted returns a State that loads/saves LastRun at path (best-effort).
func NewPersisted(path string) *State {
	s := New()
	s.SetPersistPath(path)
	return s
}

// SetPersistPath configures disk persistence and best-effort loads LastRun.
// Load failures leave LastRun unchanged / empty and never panic.
func (s *State) SetPersistPath(path string) {
	s.mu.Lock()
	s.persistPath = path
	s.mu.Unlock()
	s.loadLastRun()
}

func (s *State) loadLastRun() {
	s.mu.RLock()
	path := s.persistPath
	s.mu.RUnlock()
	if path == "" || isForbiddenPersistPath(path) {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var sum Summary
	if err := json.Unmarshal(data, &sum); err != nil {
		return
	}
	cp := sum
	cp.Accounts = append([]AccountResult(nil), sum.Accounts...)
	s.mu.Lock()
	s.lastRun = &cp
	s.mu.Unlock()
}


func (s *State) persistLastRun(summary Summary) {
	s.mu.RLock()
	path := s.persistPath
	s.mu.RUnlock()
	if path == "" || isForbiddenPersistPath(path) {
		return
	}
	data, err := json.MarshalIndent(summary, "", "  ")
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
		// Fallback: try direct write
		_ = os.WriteFile(path, data, 0o644)
	}
}

// DefaultStatePath is {CPA root}/data/codex-selective-ping/last_run.json
// where CPA root is the parent of a discovered plugins/ directory (or cwd).
func DefaultStatePath() string {
	root := resolveCPARoot()
	if root == "" {
		return ""
	}
	return filepath.Join(root, "data", pluginDataSubdir, lastRunFileName)
}

// ResolveStatePath applies optional state_path / data_dir overrides.
// Relative paths are resolved against process cwd. Empty overrides use DefaultStatePath.
func ResolveStatePath(statePath, dataDir string) string {
	if sp := strings.TrimSpace(statePath); sp != "" {
		return absAgainstCwd(sp)
	}
	if dd := strings.TrimSpace(dataDir); dd != "" {
		return filepath.Join(absAgainstCwd(dd), lastRunFileName)
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
		// If current dir itself is named plugins, parent is root.
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
