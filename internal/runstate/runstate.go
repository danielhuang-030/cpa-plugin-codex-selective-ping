package runstate

import (
	"sort"
	"strings"
	"sync"
	"time"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
	"cpa-plugin-codex-selective-ping/internal/selector"
)

type AccountResult struct {
	AuthIndex   string     `json:"auth_index,omitempty"`
	Name        string     `json:"name"`
	Email       string     `json:"email,omitempty"`
	Unavailable bool       `json:"unavailable,omitempty"`
	Status      string     `json:"status"`
	Attempts    int        `json:"attempts"`
	HTTPStatus  int        `json:"http_status,omitempty"`
	Error       string     `json:"error,omitempty"`
	EligibleAt  *time.Time `json:"eligible_at,omitempty"`
	ResetsAt    *time.Time `json:"resets_at,omitempty"`
}

type Summary struct {
	At        time.Time       `json:"at"`
	Mode      string          `json:"mode"`
	Total     int             `json:"total"`
	Attempted int             `json:"attempted"`
	Succeeded int             `json:"succeeded"`
	Failed    int             `json:"failed"`
	Limited   int             `json:"limited"`
	Skipped   int             `json:"skipped"`
	Error     string          `json:"error,omitempty"`
	Message   string          `json:"message,omitempty"`
	Accounts  []AccountResult `json:"accounts,omitempty"`
}

type AccountView struct {
	AuthIndex   string               `json:"auth_index,omitempty"`
	Name        string               `json:"name"`
	Email       string               `json:"email,omitempty"`
	Unavailable bool                 `json:"unavailable,omitempty"`
	Disabled    bool                 `json:"disabled,omitempty"`
	Selected    bool                 `json:"selected"`
	Status      string               `json:"status,omitempty"`
	Attempts    int                  `json:"attempts,omitempty"`
	LastAttempt time.Time            `json:"last_attempt,omitempty"`
	LastSuccess time.Time            `json:"last_success,omitempty"`
	EligibleAt  time.Time            `json:"eligible_at,omitempty"`
	ResetsAt    time.Time            `json:"resets_at,omitempty"`
	Error       string               `json:"error,omitempty"`
	Plan        string               `json:"plan,omitempty"`
	FiveHour    *hostapi.QuotaWindow `json:"five_hour,omitempty"`
	Weekly      *hostapi.QuotaWindow `json:"weekly,omitempty"`
}

type StatusSnapshot struct {
	Running  bool          `json:"running"`
	NextRun  *time.Time    `json:"next_run,omitempty"`
	LastRun  *Summary      `json:"last_run,omitempty"`
	Accounts []AccountView `json:"accounts,omitempty"`
}

type accountMem struct {
	Status      string
	Attempts    int
	LastAttempt time.Time
	LastSuccess time.Time
	EligibleAt  time.Time
	ResetsAt    time.Time
	Error       string
}

type State struct {
	mu          sync.RWMutex
	running     bool
	nextRun     time.Time
	lastRun     *Summary
	byIndex     map[string]accountMem
	persistPath string
}

func New() *State {
	return &State{byIndex: map[string]accountMem{}}
}

func (s *State) TryBegin() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return false
	}
	s.running = true
	return true
}

func (s *State) End(summary Summary) {
	s.mu.Lock()
	s.running = false
	cp := summary
	cp.Accounts = append([]AccountResult(nil), summary.Accounts...)
	s.lastRun = &cp
	for _, a := range summary.Accounts {
		if a.AuthIndex == "" {
			continue
		}
		m := s.byIndex[a.AuthIndex]
		m.Status = a.Status
		m.Attempts = a.Attempts
		m.Error = a.Error
		m.LastAttempt = time.Now()
		if a.EligibleAt != nil {
			m.EligibleAt = *a.EligibleAt
		}
		if a.ResetsAt != nil {
			m.ResetsAt = *a.ResetsAt
		}
		if a.Status == "success" {
			m.LastSuccess = time.Now()
			m.ResetsAt = time.Time{}
		}
		s.byIndex[a.AuthIndex] = m
	}
	toSave := cp
	s.mu.Unlock()
	s.persistLastRun(toSave)
}

func (s *State) SetNextRun(t time.Time) {
	s.mu.Lock()
	s.nextRun = t
	s.mu.Unlock()
}

type AccountMem struct {
	LastSuccess time.Time
	ResetsAt    time.Time
	EligibleAt  time.Time
	Status      string
	Attempts    int
	Error       string
}

func (s *State) GetAccount(authIndex string) AccountMem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := s.byIndex[authIndex]
	return AccountMem{LastSuccess: m.LastSuccess, ResetsAt: m.ResetsAt, EligibleAt: m.EligibleAt, Status: m.Status, Attempts: m.Attempts, Error: m.Error}
}

func (s *State) Snapshot(cfgAccounts []string, discovered []hostapi.AuthFile, nextOverride time.Time) StatusSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var next *time.Time
	nr := s.nextRun
	if !nextOverride.IsZero() {
		nr = nextOverride
	}
	if !nr.IsZero() {
		t := nr
		next = &t
	}
	var last *Summary
	if s.lastRun != nil {
		cp := *s.lastRun
		cp.Accounts = append([]AccountResult(nil), s.lastRun.Accounts...)
		last = &cp
	}
	selected := selector.Select(discovered, cfgAccounts)
	sel := map[string]bool{}
	for _, a := range selected {
		sel[a.AuthIndex] = true
	}
	views := make([]AccountView, 0, len(discovered))
	for _, f := range discovered {
		if !selector.IsCodex(f) {
			continue
		}
		m := s.byIndex[f.AuthIndex]
		views = append(views, AccountView{
			AuthIndex:   f.AuthIndex,
			Name:        safeName(f),
			Email:       f.Email,
			Unavailable: f.Unavailable,
			Disabled:    f.Disabled,
			Selected:    sel[f.AuthIndex],
			Status:      m.Status,
			Attempts:    m.Attempts,
			LastAttempt: m.LastAttempt,
			LastSuccess: m.LastSuccess,
			EligibleAt:  m.EligibleAt,
			ResetsAt:    m.ResetsAt,
			Error:       m.Error,
			Plan:        f.Plan,
			FiveHour:    f.FiveHour,
			Weekly:      f.Weekly,
		})
	}
	sort.Slice(views, func(i, j int) bool {
		return strings.ToLower(views[i].Name) < strings.ToLower(views[j].Name)
	})
	return StatusSnapshot{Running: s.running, NextRun: next, LastRun: last, Accounts: views}
}

func safeName(a hostapi.AuthFile) string {
	if a.Name != "" {
		return a.Name
	}
	if a.ID != "" {
		return a.ID
	}
	return a.AuthIndex
}
