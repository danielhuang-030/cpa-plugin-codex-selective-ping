package hostapi

import "time"

type QuotaWindow struct {
	Remaining *float64   `json:"remaining,omitempty"`
	Used      *float64   `json:"used,omitempty"`
	ResetsAt  *time.Time `json:"resets_at,omitempty"`
}

type AuthFile struct {
	ID          string       `json:"id,omitempty"`
	Name        string       `json:"name"`
	AuthIndex   string       `json:"auth_index"`
	Account     string       `json:"account,omitempty"`
	Email       string       `json:"email,omitempty"`
	Provider    string       `json:"provider,omitempty"`
	Type        string       `json:"type,omitempty"`
	Status      string       `json:"status,omitempty"`
	Disabled    bool         `json:"disabled"`
	Unavailable bool         `json:"unavailable,omitempty"`
	Plan        string       `json:"plan,omitempty"`
	FiveHour    *QuotaWindow `json:"five_hour,omitempty"`
	Weekly      *QuotaWindow `json:"weekly,omitempty"`
}
