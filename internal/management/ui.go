package management

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"cpa-plugin-codex-selective-ping/internal/runstate"
)

func nextSlotLabel(next interface{}, times []string, timezone string) string {
	hhmm := ""
	switch v := next.(type) {
	case time.Time:
		loc := loadLoc(timezone)
		hhmm = v.In(loc).Format("15:04")
	case *time.Time:
		if v != nil {
			loc := loadLoc(timezone)
			hhmm = v.In(loc).Format("15:04")
		}
	case string:
		s := strings.TrimSpace(v)
		if len(s) >= 5 && s[2] == ':' {
			hhmm = s[:5]
		} else if t, err := time.Parse(time.RFC3339, s); err == nil {
			loc := loadLoc(timezone)
			hhmm = t.In(loc).Format("15:04")
		} else {
			hhmm = s
		}
	}
	if hhmm == "" {
		return ""
	}
	for _, t := range times {
		if t == hhmm {
			return hhmm
		}
	}
	return hhmm
}

func loadLoc(timezone string) *time.Location {
	if timezone == "" {
		return time.Local
	}
	if loc, err := time.LoadLocation(timezone); err == nil {
		return loc
	}
	return time.Local
}

func lookupAccountTimes(m map[string][]string, a runstate.AccountView) ([]string, bool) {
	if len(m) == 0 {
		return nil, false
	}
	for _, key := range []string{preferID(a), a.AuthIndex, a.Email, a.Name} {
		if key == "" {
			continue
		}
		if times, ok := m[key]; ok && len(times) > 0 {
			return times, true
		}
	}
	return nil, false
}

// rematerializeAccountTimes re-keys custom times under preferID (allowlist id) so Save emits
// AuthIndex ids rather than alternate YAML/status keys (e.g. email when the card uses AuthIndex).
func rematerializeAccountTimes(accounts []runstate.AccountView, m map[string][]string) map[string][]string {
	out := map[string][]string{}
	if len(m) == 0 {
		return out
	}
	for _, a := range accounts {
		times, ok := lookupAccountTimes(m, a)
		if !ok {
			continue
		}
		id := preferID(a)
		if id == "" {
			continue
		}
		out[id] = append([]string(nil), times...)
	}
	return out
}

func modeLabel(lang Lang, mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "force", "manual":
		return T(lang, "mode_force")
	case "scheduled", "schedule":
		return T(lang, "mode_scheduled")
	default:
		if mode == "" {
			return "—"
		}
		return mode
	}
}

func renderRunHistoryList(st StatusResponse, lang Lang, t func(string) string) string {
	runs := st.RunHistory
	if len(runs) == 0 && st.LastRun != nil {
		runs = []runstate.Summary{*st.LastRun}
	}
	if len(runs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<div class="hist" data-testid="run-history">`)
	for i, r := range runs {
		openClass := ""
		chipText := t("hist_expand")
		if i == 0 {
			openClass = " open"
			chipText = t("hist_collapse")
		}
		title := fmt.Sprintf("%s · %d %s · %d %s",
			html.EscapeString(modeLabel(lang, r.Mode)),
			r.Succeeded, html.EscapeString(t("chip_ok")),
			r.Skipped, html.EscapeString(t("chip_skipped")))
		if r.Failed > 0 {
			title += fmt.Sprintf(" · %d %s", r.Failed, html.EscapeString(t("chip_failed")))
		}
		if r.Limited > 0 {
			title += fmt.Sprintf(" · %d %s", r.Limited, html.EscapeString(t("chip_limited")))
		}
		subline := fmt.Sprintf("%s · mode=%s",
			html.EscapeString(r.At.Format("2006-01-02 15:04")),
			html.EscapeString(r.Mode))
		fmt.Fprintf(&b, `<div class="hist-item%s">`, openClass)
		fmt.Fprintf(&b, `<button class="hist-head" type="button" onclick="toggleHist(this)">`)
		fmt.Fprintf(&b, `<div class="left"><div class="title">%s</div><div class="subline">%s</div></div>`, title, subline)
		fmt.Fprintf(&b, `<span class="chip" data-hist-chip>%s</span></button>`, html.EscapeString(chipText))
		b.WriteString(`<div class="hist-body" data-testid="hist-accounts">`)
		if len(r.Accounts) == 0 {
			b.WriteString(`<div class="hint">—</div>`)
		}
		for _, a := range r.Accounts {
			who := a.Name
			if who == "" {
				who = a.Email
			}
			if who == "" {
				who = a.AuthIndex
			}
			detail := fmt.Sprintf("%s · %s %d", html.EscapeString(statusLabel(lang, a.Status)), html.EscapeString(t("hist_attempts")), a.Attempts)
			if a.Error != "" {
				detail += " · " + html.EscapeString(a.Error)
			}
			fmt.Fprintf(&b, `<div class="acct-line"><div><div class="who">%s</div><div class="detail">%s</div></div><span class="chip %s">%s</span></div>`,
				html.EscapeString(who), detail, statusChipClass(a.Status), html.EscapeString(statusLabel(lang, a.Status)))
		}
		b.WriteString(`</div></div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func renderAccountCards(st StatusResponse, lang Lang, t func(string) string) (rows string, customN int) {
	var b strings.Builder
	for _, a := range st.Accounts {
		checked := ""
		cardClass := "acct-card"
		if a.Selected {
			checked = " checked"
			cardClass = "acct-card selected"
		}
		customTimes, isCustom := lookupAccountTimes(st.AccountTimes, a)
		sched := "inherit"
		if isCustom {
			sched = "custom"
			customN++
		}
		statusChip := statusChipClass(a.Status)
		statusText := orDash(a.Status)
		if !a.Selected {
			statusText = t("chip_unselected")
			statusChip = ""
		} else if a.Status == "success" || a.Status == "ok" {
			statusText = t("status_success")
			statusChip = "ok"
		}
		// Empty Status stays neutral "—" via orDash / empty chip class (not success).
		metaParts := []string{}
		if a.AuthIndex != "" {
			metaParts = append(metaParts, html.EscapeString(t("auth_index_label"))+": "+html.EscapeString(a.AuthIndex))
		}
		if a.Email != "" {
			metaParts = append(metaParts, html.EscapeString(a.Email))
		}
		if a.Plan != "" {
			metaParts = append(metaParts, "plan "+html.EscapeString(a.Plan))
		}
		meta := strings.Join(metaParts, " · ")
		id := preferID(a)
		displayName := a.Email
		if displayName == "" {
			displayName = a.Name
		}
		if displayName == "" {
			displayName = id
		}

		fmt.Fprintf(&b, `<div class="%s" data-testid="account-schedule" data-sched="%s" data-auth-index="%s" data-email="%s" data-name="%s" data-id="%s" data-selected="%t" data-custom="%t">
<div class="acct-top">
<input type="checkbox" class="acct" data-id="%s"%s onchange="syncAcctCard(this)">
<div><div class="name">%s</div><div class="meta">%s</div></div>
<span class="chip %s" data-col="status">%s</span>
</div>`,
			cardClass, sched,
			html.EscapeString(a.AuthIndex), html.EscapeString(a.Email), html.EscapeString(a.Name), html.EscapeString(id),
			a.Selected, isCustom,
			html.EscapeString(id), checked,
			html.EscapeString(displayName), meta,
			statusChip, html.EscapeString(statusText),
		)
		// light plan chip (optional)
		if a.Plan != "" {
			fmt.Fprintf(&b, `<div class="sched-row"><span class="chip" data-col="plan">%s</span></div>`, html.EscapeString(a.Plan))
		} else {
			b.WriteString(`<div class="sched-row hide-plan"><span class="chip" data-col="plan" style="display:none">—</span></div>`)
		}

		b.WriteString(`<div class="sched-row">`)
		if !a.Selected {
			fmt.Fprintf(&b, `<span class="hint">%s</span>`, html.EscapeString(t("sched_not_selected")))
		} else {
			inhOn, cusOn := "on", ""
			if isCustom {
				inhOn, cusOn = "", "on"
			}
			fmt.Fprintf(&b, `<div class="seg" role="group" aria-label="schedule">`+
				`<button type="button" class="%s" data-sched-btn="inherit" onclick="setAcctSched(this,'inherit')">%s</button>`+
				`<button type="button" class="%s" data-sched-btn="custom" onclick="setAcctSched(this,'custom')">%s</button></div>`,
				inhOn, html.EscapeString(t("sched_inherit")),
				cusOn, html.EscapeString(t("sched_custom")))
			if isCustom {
				b.WriteString(`<div class="mini-times" data-mini-times>`)
				for _, tm := range customTimes {
					fmt.Fprintf(&b, `<span class="pill" data-tm="%s">%s<button type="button" aria-label="remove" onclick="removeAcctTime(this,'%s')">×</button></span>`,
						html.EscapeString(tm), html.EscapeString(tm), html.EscapeString(tm))
				}
				fmt.Fprintf(&b, `<input type="time" class="acct-new-time" value="12:00" style="width:auto;padding:4px 8px"/>`+
					`<button class="btn-secondary" type="button" style="padding:6px 10px" onclick="addAcctTime(this)" data-i18n="sched_add">%s</button></div>`,
					html.EscapeString(t("sched_add")))
				if len(customTimes) == 0 {
					fmt.Fprintf(&b, `<span class="hint" data-testid="custom-empty-hint">%s</span>`, html.EscapeString(t("sched_custom_empty")))
				}
			} else {
				eff := strings.Join(st.Times, " · ")
				if eff == "" {
					eff = "—"
				}
				fmt.Fprintf(&b, `<span class="hint" data-effective>%s %s</span>`,
					html.EscapeString(t("sched_effective")), html.EscapeString(eff))
			}
		}
		b.WriteString(`</div></div>`)
	}
	return b.String(), customN
}

func RenderStatusPage(st StatusResponse, lang Lang) string {
	if lang == "" {
		lang = LangZhHant
	}
	t := func(key string) string { return T(lang, key) }

	next := "—"
	if st.NextRun != nil {
		switch v := st.NextRun.(type) {
		case time.Time:
			next = v.Format(time.RFC3339)
		case *time.Time:
			if v != nil {
				next = v.Format(time.RFC3339)
			}
		case string:
			if v != "" {
				next = v
			}
		}
	}
	enabled := t("enabled_off")
	if st.Enabled {
		enabled = t("enabled_on")
	}
	enabledChecked := ""
	if st.Enabled {
		enabledChecked = " checked"
	}
	nextSlot := nextSlotLabel(st.NextRun, st.Times, st.Timezone)
	railNext := next
	if nextSlot != "" {
		railNext = nextSlot
	}
	nextInGlobal := false
	for _, tm := range st.Times {
		if tm == nextSlot {
			nextInGlobal = true
			break
		}
	}
	customNextNote := ""
	if nextSlot != "" && !nextInGlobal {
		customNextNote = fmt.Sprintf(`<p class="hint" data-testid="next-custom-only">%s</p>`,
			html.EscapeString(fmt.Sprintf(t("next_custom_only"), nextSlot)))
	}
	nowHHMM := time.Now().In(loadLoc(st.Timezone)).Format("15:04")
	var timeline strings.Builder
	for _, tm := range st.Times {
		slotClass, captionKey := rhythmSlot(tm, nextSlot, nowHHMM)
		caption := ""
		if captionKey == "slot_next" {
			caption = t("slot_next_inherit")
		} else if captionKey != "" {
			caption = t(captionKey)
		}
		fmt.Fprintf(&timeline, `<div class="%s" data-time="%s"><button class="x" type="button" aria-label="remove" onclick="removeTime('%s')">×</button><div class="t">%s</div><div class="caption">%s</div></div>`,
			slotClass, html.EscapeString(tm), html.EscapeString(tm), html.EscapeString(tm), html.EscapeString(caption))
	}
	selectedN := 0
	for _, a := range st.Accounts {
		if a.Selected {
			selectedN++
		}
	}
	rows, customN := renderAccountCards(st, lang, t)
	accountsEmptyClass := ""
	accountsFilledClass := ""
	if len(st.Accounts) == 0 {
		accountsFilledClass = " hidden"
		accountsEmptyClass = ""
	} else {
		accountsEmptyClass = " hidden"
	}
	lastFilledClass := " hidden"
	lastEmptyClass := ""
	lastBlock := ""
	if st.LastRun != nil || len(st.RunHistory) > 0 {
		lastFilledClass = ""
		lastEmptyClass = " hidden"
		lr := st.LastRun
		if lr == nil && len(st.RunHistory) > 0 {
			lr = &st.RunHistory[0]
		}
		big := "—"
		metaLine := ""
		if lr != nil {
			big = fmt.Sprintf("%d %s", lr.Succeeded, t("chip_ok"))
			metaLine = fmt.Sprintf("%s · %s · %s %d · %s %d",
				html.EscapeString(lr.Mode),
				html.EscapeString(lr.At.Format("15:04")),
				html.EscapeString(t("chip_failed")), lr.Failed,
				html.EscapeString(t("chip_skipped")), lr.Skipped)
			if lr.Limited > 0 {
				metaLine += fmt.Sprintf(" · %s %d", html.EscapeString(t("chip_limited")), lr.Limited)
			}
		}
		hist := renderRunHistoryList(st, lang, t)
		lastBlock = fmt.Sprintf(`<div class="run"><div class="run-summary"><div class="label" data-i18n="last_run_receipt_label">%s</div><div class="big">%s</div><div class="meta">%s</div></div>%s</div>`,
			html.EscapeString(t("last_run_receipt_label")), html.EscapeString(big), metaLine, hist)
		if lr != nil && lr.Message != "" {
			lastBlock += `<p class="hint">` + html.EscapeString(lr.Message) + `</p>`
		}
	}
	timesJSON, _ := json.Marshal(st.Times)
	seededAccountTimes := rematerializeAccountTimes(st.Accounts, st.AccountTimes)
	accountTimesJSON, _ := json.Marshal(seededAccountTimes)
	if len(seededAccountTimes) == 0 {
		accountTimesJSON = []byte("{}")
	}
	banner := fmt.Sprintf(t("banner_selected"), selectedN, len(st.Accounts))
	if len(st.AccountsConfig) == 0 {
		banner = t("banner_empty")
	}

	needKey := t("need_key")
	saving := t("saving")
	starting := t("starting")
	runPolling := t("run_polling")
	runAlready := t("run_already")

	runningAttr := "false"
	runDisabled := ""
	runningBadgeHidden := " hidden"
	runButtonLabel := t("run_now")
	runIdleLabelAttr := fmt.Sprintf(` data-idle-label="%s"`, html.EscapeString(t("run_now")))
	if st.Running {
		runningAttr = "true"
		runDisabled = " disabled"
		runningBadgeHidden = ""
		runButtonLabel = t("running_label")
	}
	runNowAttrs := runDisabled + runIdleLabelAttr

	return fmt.Sprintf(`<!doctype html>
<html lang="%s">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>%s</title>
<style>
  :root, :root[data-theme="light"] {
    --bg: #f6f1ea;
    --bg-2: #efe7dc;
    --ink: #1a1523;
    --ink-soft: #5b534c;
    --line: #e0d5c8;
    --panel: #fffaf4;
    --panel-2: #f3ebe1;
    --accent: #c45c26;
    --accent-ink: #fff7f0;
    --accent-2: #2f5d50;
    --ok: #1f7a4c; --ok-bg: #e5f6ec;
    --warn: #a15c12; --warn-bg: #fff0d9;
    --bad: #a11f2c; --bad-bg: #fde8ea;
    --chip: #ece3d7;
    --shadow: 0 18px 40px rgba(60, 40, 20, .08);
    --mono: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    --sans: "Iowan Old Style", "Palatino Linotype", Palatino, "Book Antiqua", Georgia, serif;
    --ui: ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif;
  }
  :root[data-theme="dark"] {
    --bg: #141018; --bg-2: #1c1620; --ink: #f4efe8; --ink-soft: #b9aea3;
    --line: #3a3038; --panel: #1e1822; --panel-2: #261f2b;
    --accent: #e08a52; --accent-ink: #1a1010; --accent-2: #7dbaa8;
    --ok: #6dcaa0; --ok-bg: #1b3328; --warn: #e2b16a; --warn-bg: #3a2a14;
    --bad: #f0a0a8; --bad-bg: #3a1a20; --chip: #2c2430;
    --shadow: 0 18px 40px rgba(0,0,0,.35);
  }
  * { box-sizing: border-box }
  body {
    margin: 0; background: var(--bg); color: var(--ink);
    font-family: var(--ui); line-height: 1.45;
  }
  .shell {
    max-width: 1180px; margin: 0 auto; padding: 20px 16px 48px;
    display: grid; grid-template-columns: 280px 1fr; gap: 18px;
  }
  @media (max-width: 960px) {
    .shell { grid-template-columns: 1fr }
    .rail { position: static; top: auto; }
  }
  .rail {
    background: var(--panel);
    border: 1px solid var(--line);
    border-radius: 22px;
    padding: 18px 16px 20px;
    box-shadow: var(--shadow);
    align-self: start;
    position: sticky; top: 14px;
  }
  .brand {
    font-family: var(--sans);
    font-size: 28px; line-height: 1.05; letter-spacing: -.03em;
    margin: 0 0 6px;
  }
  .brand em { font-style: italic; color: var(--accent); font-weight: 600 }
  .lede { margin: 0 0 16px; color: var(--ink-soft); font-size: 13px }
  .rail-block { margin-top: 16px; padding-top: 14px; border-top: 1px dashed var(--line) }
  .rail-block h3 {
    margin: 0 0 8px; font-size: 11px; letter-spacing: .08em;
    text-transform: uppercase; color: var(--ink-soft); font-weight: 700;
  }
  .metric {
    display:flex; justify-content:space-between; align-items:baseline;
    padding: 8px 0; border-bottom: 1px solid var(--line);
    font-variant-numeric: tabular-nums;
  }
  .metric:last-child { border-bottom: 0 }
  .metric .k { color: var(--ink-soft); font-size: 12px }
  .metric .v { font-weight: 700; font-size: 14px }
  .status-pill {
    display:inline-flex; align-items:center; gap:6px;
    padding: 6px 10px; border-radius: 999px;
    background: var(--ok-bg); color: var(--ok); font-size: 12px; font-weight: 700;
  }
  .status-pill.warn, .status-pill.off, .status-pill.neutral {
    background: var(--chip); color: var(--ink-soft);
  }
  .status-pill i {
    width:8px; height:8px; border-radius:50%%; background: currentColor; display:inline-block;
  }
  .primary-stack { display:grid; gap:8px; margin-top: 14px }
  button, .btn {
    font: inherit; cursor: pointer; border-radius: 12px; border: 0;
    padding: 10px 12px; font-weight: 700; font-size: 13px;
  }
  .btn-primary { background: var(--accent); color: var(--accent-ink) }
  .btn-secondary { background: var(--panel-2); color: var(--ink); border: 1px solid var(--line) }
  .btn-ghost { background: transparent; color: var(--ink-soft); border: 1px dashed var(--line) }
  .btn-primary:hover { filter: brightness(1.06) }
  .btn-primary:active { transform: translateY(1px); filter: brightness(.96) }
  .btn-secondary:hover { background: color-mix(in srgb, var(--ink) 6%%, var(--panel-2)) }
  .btn-secondary:active { transform: translateY(1px) }
  .btn-ghost:hover { color: var(--ink); border-style: solid }
  .btn-ghost:active { transform: translateY(1px) }
  button:disabled, .btn:disabled,
  .btn-primary:disabled, .btn-secondary:disabled, .btn-ghost:disabled {
    opacity: .55; cursor: not-allowed; filter: none; transform: none;
  }
  button:disabled:hover, button:disabled:active,
  .btn-primary:disabled:hover, .btn-primary:disabled:active,
  .btn-secondary:disabled:hover, .btn-secondary:disabled:active,
  .btn-ghost:disabled:hover, .btn-ghost:disabled:active {
    transform: none; filter: none;
  }
  .field { display:grid; gap:6px; font-size:12px; color: var(--ink-soft); font-weight: 650 }
  input[type=text], input[type=password], input[type=time] {
    border: 1px solid var(--line); background: var(--bg);
    border-radius: 11px; padding: 10px 11px; color: var(--ink); font: inherit; width: 100%%;
  }
  .workspace { display:grid; gap: 16px }
  .hero {
    display:grid; grid-template-columns: 1.35fr .65fr; gap: 14px;
  }
  @media (max-width: 960px) { .hero { grid-template-columns: 1fr } }
  .panel {
    background: var(--panel);
    border: 1px solid var(--line);
    border-radius: 22px;
    padding: 18px;
    box-shadow: var(--shadow);
  }
  .panel h2 {
    margin: 0 0 4px;
    font-family: var(--sans);
    font-size: 22px; letter-spacing: -.02em;
  }
  .panel .sub { margin: 0 0 14px; color: var(--ink-soft); font-size: 13px }
  .timeline {
    display:grid; grid-template-columns: repeat(4, 1fr); gap: 8px;
  }
  @media (max-width: 800px) { .timeline { grid-template-columns: repeat(2, 1fr) } }
  .slot {
    background: var(--bg-2);
    border: 1px solid var(--line);
    border-radius: 16px;
    padding: 12px 12px 14px;
    position: relative;
  }
  .slot .t {
    font-family: var(--mono); font-size: 20px; font-weight: 700;
    font-variant-numeric: tabular-nums; letter-spacing: -.03em;
  }
  .slot .caption { margin-top: 4px; font-size: 11px; color: var(--ink-soft) }
  .slot .x {
    position:absolute; top:8px; right:10px; border:0; background:transparent;
    color: var(--ink-soft); font-size: 16px; padding: 0; cursor:pointer;
  }
  .slot.next { outline: 2px solid var(--accent); background: color-mix(in srgb, var(--accent) 12%%, var(--panel)) }
  .slot.next .caption { color: var(--accent); font-weight: 700 }
  .add-row { display:flex; gap:8px; flex-wrap:wrap; align-items:end; margin-top: 12px }
  .accounts-head {
    display:flex; justify-content:space-between; gap:12px; align-items:end; flex-wrap:wrap;
    margin-bottom: 12px;
  }
  .tabs { display:flex; gap:6px; flex-wrap:wrap }
  .tab {
    border:1px solid var(--line); background: var(--bg-2); color: var(--ink-soft);
    border-radius: 999px; padding: 6px 11px; font-size: 12px; font-weight: 700;
  }
  .tab.on { background: var(--ink); color: var(--panel); border-color: var(--ink) }
  .account-grid { display:grid; gap: 12px }
  .acct-card {
    display:grid; gap: 12px;
    padding: 14px 16px;
    border-radius: 18px;
    border: 1px solid var(--line);
    background: var(--bg);
  }
  .acct-card.selected {
    background: color-mix(in srgb, var(--accent-2) 10%%, var(--panel));
    border-color: color-mix(in srgb, var(--accent-2) 45%%, var(--line));
  }
  .acct-top { display:grid; grid-template-columns: 28px 1fr auto; gap: 10px; align-items: center }
  .acct-card .name { font-weight: 750; font-size: 14px }
  .acct-card .meta { color: var(--ink-soft); font-size: 11px; margin-top: 2px; font-family: var(--mono) }
  .sched-row { display:flex; flex-wrap:wrap; gap:10px; align-items:center; padding-top:4px; border-top:1px dashed var(--line) }
  .seg { display:inline-flex; border:1px solid var(--line); border-radius:999px; overflow:hidden; background:var(--panel) }
  .seg button { border:0; background:transparent; padding:6px 12px; font-size:12px; font-weight:700; color:var(--ink-soft); border-radius:0 }
  .seg button.on { background:var(--ink); color:var(--panel) }
  .mini-times { display:flex; flex-wrap:wrap; gap:6px; align-items:center }
  .mini-times .pill { font-family:var(--mono); font-size:12px; font-weight:700; padding:4px 8px; border-radius:999px; background:var(--panel-2); border:1px solid var(--line) }
  .mini-times .pill button { border:0; background:transparent; color:var(--ink-soft); margin-left:4px; cursor:pointer; padding:0; font-size:12px }
  .chip {
    display:inline-flex; align-items:center; padding: 4px 9px; border-radius: 999px;
    font-size: 11px; font-weight: 750;
    background: var(--chip); color: var(--ink);
  }
  .chip.ok { background: var(--ok-bg); color: var(--ok) }
  .chip.warn { background: var(--warn-bg); color: var(--warn) }
  .chip.bad { background: var(--bad-bg); color: var(--bad) }
  .running-banner {
    display:flex; flex-direction:column; gap:4px;
    margin: 10px 0 4px;
    padding: 12px 14px;
    border-radius: 14px;
    background: var(--warn-bg);
    color: var(--warn);
    border: 1px solid color-mix(in srgb, var(--warn) 35%%, var(--line));
    font-family: var(--ui);
    font-size: 13px;
    font-weight: 600;
    line-height: 1.35;
  }
  .running-banner strong { font-size: 14px; letter-spacing: .02em; }
  .running-banner[hidden] { display:none !important; }
  .run {
    display:grid; grid-template-columns: 170px 1fr; gap: 14px;
  }
  @media (max-width: 800px) { .run { grid-template-columns: 1fr } }
  .run-summary {
    background: var(--ink); color: var(--panel);
    border-radius: 18px; padding: 16px;
  }
  .run-summary .label { font-size: 11px; letter-spacing: .08em; text-transform: uppercase; opacity: .7 }
  .run-summary .big {
    font-family: var(--sans); font-size: 30px; margin: 8px 0 4px; letter-spacing: -.03em;
  }
  .run-summary .meta { font-size: 12px; opacity: .75; font-family: var(--mono) }
  .hist { display:grid; gap: 8px }
  .hist-item { border:1px solid var(--line); border-radius:14px; background:var(--bg); overflow:hidden }
  .hist-head { display:flex; justify-content:space-between; gap:10px; align-items:center; padding:10px 12px; cursor:pointer; width:100%%; border:0; background:transparent; color:inherit; text-align:left; font:inherit }
  .hist-head:hover { background: color-mix(in srgb, var(--accent) 6%%, transparent) }
  .hist-head .left { display:grid; gap:2px }
  .hist-head .title { font-weight:750; font-size:13px }
  .hist-head .subline { font-size:11px; color:var(--ink-soft); font-family:var(--mono) }
  .hist-body { display:none; padding:0 12px 12px; border-top:1px dashed var(--line) }
  .hist-item.open .hist-body { display:grid; gap:6px; padding-top:10px }
  .acct-line { display:flex; justify-content:space-between; gap:8px; align-items:center; padding:8px 10px; border-radius:12px; background:var(--panel); border:1px solid var(--line); font-size:12px }
  .acct-line .who { font-weight:700 }
  .acct-line .detail { color:var(--ink-soft); font-family:var(--mono); font-size:11px }
  .empty {
    border: 1.5px dashed var(--line);
    border-radius: 18px; padding: 22px 18px; background: var(--bg-2);
  }
  .empty strong { display:block; font-family: var(--sans); font-size: 20px; margin-bottom: 6px }
  .empty p { margin: 0 0 12px; color: var(--ink-soft); font-size: 13px }
  .hidden { display:none !important }
  .row { display:flex; gap:8px; flex-wrap:wrap; align-items:center }
  .foot { text-align:center; color: var(--ink-soft); font-size: 11px; margin-top: 8px }
  .lang-switch{display:flex;gap:6px;flex-wrap:wrap;margin:8px 0 12px}
  .lang-switch a{color:var(--ink-soft);text-decoration:none;font-size:12px;padding:4px 8px;border-radius:999px;border:1px solid var(--line)}
  .lang-switch a.active,.lang-switch a:hover{background:var(--ink);color:var(--panel);border-color:var(--ink)}
  .banner{background:color-mix(in srgb, var(--warn) 14%%, var(--panel));border:1px solid var(--line);color:var(--warn);border-radius:12px;padding:10px 12px;font-size:13px;margin:8px 0 12px}
  pre{white-space:pre-wrap;background:var(--bg-2);padding:12px;border-radius:12px;font-size:12px;max-height:160px;overflow:auto}
  .hint{font-size:11px;color:var(--ink-soft)}
  @media (prefers-reduced-motion: reduce) {
    * { transition: none !important; animation: none !important }
  }
</style>
</head>
<body data-running="%s">
<div class="shell">
<aside class="rail">
  <p class="brand">Selective<br/><em>Ping</em></p>
  <p class="lede" data-i18n="subtitle">%s</p>
  <nav class="lang-switch" aria-label="language">
    <a href="?lang=zh-Hant" class="%s" data-lang="zh-Hant">%s</a>
    <a href="?lang=en" class="%s" data-lang="en">%s</a>
    <a href="?lang=ja" class="%s" data-lang="ja">%s</a>
  </nav>
  <div class="status-pill %s"><i></i> %s · v%s</div>
  <span id="running-badge" class="chip warn" data-i18n="running_label"%s>%s</span>
  <div id="running-banner" class="running-banner" data-testid="running-banner"%s>
    <strong data-i18n="running_label">%s</strong>
    <span data-i18n="run_polling">%s</span>
  </div>
  <div class="rail-block">
    <h3 data-i18n="rail_now">%s</h3>
    <div class="metric"><span class="k" data-i18n="next_run">%s</span><span class="v">%s</span></div>
    <div class="metric"><span class="k" data-i18n="rail_whitelist">%s</span><span class="v">%d / %d</span></div>
    <div class="metric"><span class="k" data-i18n="rail_model">%s</span><span class="v">%s</span></div>
    <div class="metric"><span class="k" data-i18n="timezone">%s</span><span class="v">%s</span></div>
  </div>
  <div class="rail-block">
    <h3 data-i18n="rail_key">%s</h3>
    <label class="field">Management Key
      <input id="management-key" type="password" autocomplete="off" data-i18n-placeholder="key_placeholder" placeholder="%s"/>
    </label>
    <div class="primary-stack">
      <button class="btn-primary" type="button" onclick="saveCfg()" data-i18n="save">%s</button>
      <button class="btn-secondary" type="button" onclick="runNow()" data-run-now%s data-i18n="run_now">%s</button>
      <button class="btn-ghost" type="button" onclick="location.reload()" data-i18n="refresh">%s</button>
    </div>
    <p class="hint" data-i18n="actions_hint">%s</p>
    <pre id="result"></pre>
  </div>
</aside>
<div class="workspace">
<section class="hero">
<section class="panel" id="sec-rhythm">
<h2 data-i18n="rhythm_title">%s</h2>
<p class="sub" data-i18n="rhythm_sub">%s</p>
<div class="timeline" id="times">%s</div>
%s
<div class="add-row">
  <label class="field" data-i18n="timezone">%s<input id="tz" type="text" value="%s"/></label>
  <label class="field" data-i18n="add_time">%s<input id="new-time" type="time" value="21:00"/></label>
  <button class="btn-secondary" type="button" onclick="addTime()" data-i18n="add">%s</button>
  <label style="display:flex;gap:8px;align-items:center;font-size:13px;font-weight:700;margin-left:auto">
    <input id="schedule_enabled" type="checkbox"%s/> <span data-i18n="enable">%s</span>
  </label>
</div>
</section>
<section class="panel" id="sec-principles" style="background: linear-gradient(160deg, color-mix(in srgb, var(--accent) 16%%, var(--panel)), var(--panel));">
<h2 data-i18n="how_title">%s</h2>
<p class="sub" data-i18n="how_body">%s</p>
<div class="metric"><span class="k" data-i18n="how_global_off">%s</span><span class="v" data-i18n="how_global_off_v">%s</span></div>
<div class="metric"><span class="k" data-i18n="how_manual">%s</span><span class="v" data-i18n="how_manual_v">%s</span></div>
<div class="metric"><span class="k" data-i18n="how_quota_removed">%s</span><span class="v" data-i18n="how_quota_removed_v">%s</span></div>
<div class="metric"><span class="k" data-i18n="status">%s</span><span class="v"><span class="chip %s">%s</span></span></div>
</section>
</section>
<section class="panel%s" id="sec-accounts-filled">
<div class="accounts-head">
  <div>
    <h2 data-i18n="accounts_who_title">%s</h2>
    <p class="sub" data-i18n="accounts_times_sub" style="margin:0">%s</p>
  </div>
  <div class="tabs">
    <button class="tab on" type="button" data-filter="all" onclick="filterAccounts('all')" data-i18n="filter_all">%s</button>
    <button class="tab" type="button" data-filter="selected" onclick="filterAccounts('selected')" data-i18n="filter_selected">%s</button>
    <button class="tab" type="button" data-filter="custom" onclick="filterAccounts('custom')" data-i18n="filter_custom">%s</button>
  </div>
</div>
<div class="row" style="margin:0 0 12px;gap:8px">
  <button class="btn-secondary" type="button" onclick="setAll(true)" data-i18n="select_all">%s</button>
  <button class="btn-secondary" type="button" onclick="setAll(false)" data-i18n="clear">%s</button>
</div>
<div class="banner" id="banner">%s</div>
<div class="account-grid" id="account-grid">%s</div>
</section>
<section class="panel%s" id="sec-accounts-empty">
  <h2 data-i18n="accounts_none_title">%s</h2>
  <div class="empty">
    <strong data-i18n="accounts_empty_title">%s</strong>
    <p data-i18n="accounts_none_body">%s</p>
    <button class="btn-primary" type="button" onclick="location.reload()" data-i18n="accounts_none_cta">%s</button>
  </div>
</section>
<section class="panel%s" id="sec-last-filled">
<h2 data-i18n="run_history_title">%s</h2>
<p class="sub" data-i18n="run_history_sub">%s</p>
%s
</section>
<section class="panel%s" id="sec-last-empty">
<h2 data-i18n="run_history_title">%s</h2>
<div class="empty">
<strong data-i18n="last_run_empty_title">%s</strong>
<p data-i18n="last_run_empty_body">%s</p>
<button class="btn-secondary" type="button" onclick="runNow()" data-run-now%s data-i18n="run_now">%s</button>
</div>
</section>
</div>
</div>
<script>
const initialTimes = %s;
const initialAccountTimes = %s;
const nextSlotHint = %q;
const slotCaptionNext = %q;
const slotCaptionPast = %q;
const labelSchedEffective = %q;
const labelSchedNotSelected = %q;
const labelSchedInherit = %q;
const labelSchedCustom = %q;
const labelSchedCustomEmpty = %q;
const labelHistExpand = %q;
const labelHistCollapse = %q;
const pluginId = "codex-selective-ping";
const serverLang = %q;
const msgNeedKey = %q;
const msgSaving = %q;
const msgStarting = %q;
const msgRunPolling = %q;
const msgRunAlready = %q;
const msgRunningLabel = %q;
const POLL_MS = 2500;
const STATUS_URL = '/v0/management/plugins/codex-selective-ping/status';
const RUN_URL = '/v0/management/plugins/codex-selective-ping/run';
let pollTimer = null;
let times = Array.isArray(initialTimes) ? initialTimes.slice() : [];
let accountTimes = (initialAccountTimes && typeof initialAccountTimes === 'object') ? JSON.parse(JSON.stringify(initialAccountTimes)) : {};
let accountSched = {};
function rematerializeAccountTimesFromCards(){
  document.querySelectorAll('#account-grid > [data-testid="account-schedule"]').forEach(function(card){
    const id = card.getAttribute('data-id');
    if(!id) return;
    accountSched[id] = card.getAttribute('data-sched') || 'inherit';
    const keys = [id, card.getAttribute('data-auth-index'), card.getAttribute('data-email'), card.getAttribute('data-name')].filter(Boolean);
    let found = null;
    for (let i=0;i<keys.length;i++){
      const k = keys[i];
      if(accountTimes[k] && accountTimes[k].length){ found = accountTimes[k].slice(); break; }
    }
    if(!found || !found.length){
      const pills = Array.from(card.querySelectorAll('[data-tm]')).map(function(p){ return p.getAttribute('data-tm'); }).filter(Boolean);
      if(pills.length) found = pills;
    }
    keys.forEach(function(k){ if(k !== id) delete accountTimes[k]; });
    if(accountSched[id] === 'custom'){
      accountTimes[id] = found ? found.slice() : (accountTimes[id] || []);
    } else {
      delete accountTimes[id];
    }
  });
  // Drop any leftover alternate keys not matching a card data-id.
  const allow = {};
  document.querySelectorAll('#account-grid > [data-testid="account-schedule"]').forEach(function(card){
    const id = card.getAttribute('data-id');
    if(id) allow[id] = true;
  });
  Object.keys(accountTimes).forEach(function(k){ if(!allow[k]) delete accountTimes[k]; });
}
rematerializeAccountTimesFromCards();

function normalizeCPALang(raw){
  if(!raw) return '';
  let s = String(raw).trim();
  if(s.charAt(0) === '{'){
    try{
      const j = JSON.parse(s);
      if(j && j.state && j.state.language) s = String(j.state.language);
      else if(j && j.language) s = String(j.language);
    }catch(e){}
  }
  s = String(s).trim();
  const lower = s.toLowerCase();
  if(['zh-tw','zh-hant','zh-hk','zh-mo','zh_tw','zh_hant','zh_hk','zh_mo'].indexOf(lower)>=0) return 'zh-Hant';
  if(lower === 'en' || lower.indexOf('en-')===0 || lower.indexOf('en_')===0) return 'en';
  if(lower === 'ja' || lower.indexOf('ja-')===0 || lower.indexOf('ja_')===0) return 'ja';
  return 'zh-Hant';
}

(function alignCPALanguage(){
  try{
    const params = new URLSearchParams(location.search);
    if(params.has('lang')) return;
    const raw = localStorage.getItem('cli-proxy-language');
    if(raw == null || raw === '') return;
    const mapped = normalizeCPALang(raw);
    if(mapped && mapped !== serverLang){
      params.set('lang', mapped);
      const q = params.toString();
      location.replace(location.pathname + (q ? ('?' + q) : '') + location.hash);
    }
  }catch(e){}
})();

(function alignCPATheme(){
  function normalizeTheme(raw){
    if(raw == null || raw === '') return '';
    let s = String(raw).trim();
    if(s.charAt(0) === '{'){
      try{
        const j = JSON.parse(s);
        if(j && j.state && j.state.theme) s = String(j.state.theme);
        else if(j && j.theme) s = String(j.theme);
        else if(j && j.colorScheme) s = String(j.colorScheme);
      }catch(e){}
    }
    s = String(s).trim().toLowerCase();
    if(s === 'dark' || s.indexOf('dark') >= 0) return 'dark';
    if(s === 'light' || s.indexOf('light') >= 0) return 'light';
    return '';
  }
  function readStoredTheme(){
    const keys = ['cli-proxy-theme','cli-proxy-color-scheme','theme'];
    for(let i=0;i<keys.length;i++){
      try{
        const v = localStorage.getItem(keys[i]);
        const n = normalizeTheme(v);
        if(n) return n;
      }catch(e){}
    }
    return '';
  }
  function detectDomTheme(){
    try{
      const el = document.documentElement;
      if(el && el.dataset && el.dataset.theme){
        const n = normalizeTheme(el.dataset.theme);
        if(n) return n;
      }
      if(el && el.classList){
        if(el.classList.contains('dark')) return 'dark';
        if(el.classList.contains('light')) return 'light';
      }
    }catch(e){}
    return '';
  }
  function detectPrefers(){
    try{
      if(window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) return 'dark';
    }catch(e){}
    return 'light';
  }
  try{
    const params = new URLSearchParams(location.search);
    let theme = '';
    if(params.has('theme')) theme = normalizeTheme(params.get('theme'));
    if(!theme) theme = readStoredTheme();
    if(!theme) theme = detectDomTheme();
    if(!theme) theme = detectPrefers();
    if(theme !== 'dark' && theme !== 'light') theme = 'light';
    document.documentElement.setAttribute('data-theme', theme);
  }catch(e){
    try{ document.documentElement.setAttribute('data-theme', 'light'); }catch(_){}
  }
})();

document.querySelectorAll('.lang-switch a[data-lang]').forEach(function(a){
  a.addEventListener('click', function(e){
    e.preventDefault();
    var u=new URL(location.href);
    u.searchParams.set('lang', a.getAttribute('data-lang'));
    location.assign(u.toString());
  });
});
function nowHHMM(){
  try{
    const tz=(document.getElementById('tz')||{}).value||undefined;
    const parts=new Intl.DateTimeFormat('en-GB',{timeZone:tz,hour:'2-digit',minute:'2-digit',hour12:false}).formatToParts(new Date());
    const hh=(parts.find(p=>p.type==='hour')||{}).value||'00';
    const mi=(parts.find(p=>p.type==='minute')||{}).value||'00';
    return String(hh).padStart(2,'0')+':'+String(mi).padStart(2,'0');
  }catch(e){
    const d=new Date();
    return String(d.getHours()).padStart(2,'0')+':'+String(d.getMinutes()).padStart(2,'0');
  }
}
function renderTimes(){
  const el = document.getElementById('times');
  if(!el) return;
  el.innerHTML = '';
  const now = nowHHMM();
  times.forEach((tm)=>{
    const d = document.createElement('div');
    const isNext = nextSlotHint && tm === nextSlotHint;
    const isPast = !isNext && tm < now;
    d.className = isNext ? 'slot next' : 'slot';
    d.setAttribute('data-time', tm);
    const x = document.createElement('button');
    x.type='button'; x.className='x'; x.setAttribute('aria-label','remove'); x.textContent='×';
    x.onclick=()=>removeTime(tm);
    const tEl = document.createElement('div'); tEl.className='t'; tEl.textContent=tm;
    const c = document.createElement('div'); c.className='caption';
    c.textContent = isNext ? slotCaptionNext : (isPast ? slotCaptionPast : '');
    d.appendChild(x); d.appendChild(tEl); d.appendChild(c); el.appendChild(d);
  });
  document.querySelectorAll('[data-effective]').forEach(function(h){
    h.textContent = labelSchedEffective + ' ' + (times.length ? times.join(' · ') : '—');
  });
}
function removeTime(tm){
  const i = times.indexOf(tm);
  if(i>=0){ times.splice(i,1); renderTimes(); }
}
function addTime(){
  const v = document.getElementById('new-time').value;
  if(!v) return;
  const n = v.slice(0,5);
  if(!times.includes(n)) times.push(n);
  times.sort(); renderTimes();
}
let acctFilter='all';
function setAll(v){
  document.querySelectorAll('input.acct').forEach(c=>{ c.checked=v; syncAcctCard(c, true); });
  filterAccounts(acctFilter);
}
function syncAcctCard(cb, skipFilter){
  const card=cb.closest('.acct-card[data-auth-index],div[data-testid="account-schedule"]');
  if(!card) return;
  card.classList.toggle('selected', cb.checked);
  card.setAttribute('data-selected', cb.checked ? 'true' : 'false');
  rebuildAcctSchedRow(card, cb.checked);
  if(!skipFilter) filterAccounts(acctFilter);
}
function rebuildAcctSchedRow(card, selected){
  const id=cardId(card);
  let row=card.querySelector('.sched-row:last-of-type');
  if(!row) return;
  row.innerHTML='';
  if(!selected){
    const hint=document.createElement('span');
    hint.className='hint';
    hint.textContent=labelSchedNotSelected;
    row.appendChild(hint);
    return;
  }
  const mode=accountSched[id]||'inherit';
  const seg=document.createElement('div');
  seg.className='seg';
  seg.setAttribute('role','group');
  seg.setAttribute('aria-label','schedule');
  ['inherit','custom'].forEach(function(m){
    const b=document.createElement('button');
    b.type='button';
    b.setAttribute('data-sched-btn', m);
    if(m===mode) b.className='on';
    b.textContent = m==='inherit' ? labelSchedInherit : labelSchedCustom;
    b.onclick=function(){ setAcctSched(b, m); };
    seg.appendChild(b);
  });
  row.appendChild(seg);
  if(mode==='inherit'){
    delete accountTimes[id];
    const hint=document.createElement('span');
    hint.className='hint';
    hint.setAttribute('data-effective','');
    hint.textContent=labelSchedEffective+' '+(times.length?times.join(' · '):'—');
    row.appendChild(hint);
  } else {
    if(!accountTimes[id]) accountTimes[id]=[];
    renderMiniTimes(card, row);
    if(!(accountTimes[id]&&accountTimes[id].length)){
      const eh=document.createElement('span');
      eh.className='hint';
      eh.setAttribute('data-testid','custom-empty-hint');
      eh.textContent=labelSchedCustomEmpty;
      row.appendChild(eh);
    }
  }
}
function toggleHist(btn){
  const item=btn.parentElement;
  if(!item) return;
  item.classList.toggle('open');
  const chip=btn.querySelector('[data-hist-chip],.chip');
  if(chip) chip.textContent = item.classList.contains('open') ? labelHistCollapse : labelHistExpand;
}
function filterAccounts(mode){
  if(mode) acctFilter=mode;
  document.querySelectorAll('.tabs .tab').forEach(t=>t.classList.toggle('on', t.getAttribute('data-filter')===acctFilter));
  document.querySelectorAll('#account-grid > [data-testid="account-schedule"]').forEach(card=>{
    const sel=card.getAttribute('data-selected')==='true';
    const custom=card.getAttribute('data-custom')==='true' || card.getAttribute('data-sched')==='custom';
    let show=true;
    if(acctFilter==='selected') show=sel;
    if(acctFilter==='custom') show=custom;
    card.style.display = show ? '' : 'none';
  });
}
function cardId(card){ return card.getAttribute('data-id') || ''; }
function setAcctSched(btn, mode){
  const card=btn.closest('[data-testid="account-schedule"]');
  if(!card) return;
  const id=cardId(card);
  accountSched[id]=mode;
  card.setAttribute('data-sched', mode);
  card.setAttribute('data-custom', mode==='custom' ? 'true' : 'false');
  card.querySelectorAll('[data-sched-btn]').forEach(b=>b.classList.toggle('on', b.getAttribute('data-sched-btn')===mode));
  let row=card.querySelector('.sched-row:last-of-type');
  if(!row) return;
  // keep seg; replace trailing content after seg
  const seg=row.querySelector('.seg');
  row.innerHTML='';
  if(seg) row.appendChild(seg);
  if(mode==='inherit'){
    delete accountTimes[id];
    const hint=document.createElement('span');
    hint.className='hint';
    hint.setAttribute('data-effective','');
    hint.textContent=labelSchedEffective+' '+(times.length?times.join(' · '):'—');
    row.appendChild(hint);
  } else {
    if(!accountTimes[id]) accountTimes[id]=[];
    renderMiniTimes(card, row);
  }
  filterAccounts(acctFilter);
}
function renderMiniTimes(card, row){
  const id=cardId(card);
  let box=row.querySelector('[data-mini-times]');
  if(!box){
    box=document.createElement('div');
    box.className='mini-times';
    box.setAttribute('data-mini-times','');
    row.appendChild(box);
  }
  box.innerHTML='';
  (accountTimes[id]||[]).forEach(function(tm){
    const pill=document.createElement('span');
    pill.className='pill';
    pill.setAttribute('data-tm', tm);
    pill.textContent=tm;
    const x=document.createElement('button');
    x.type='button'; x.setAttribute('aria-label','remove'); x.textContent='×';
    x.onclick=function(){ removeAcctTime(x, tm); };
    pill.appendChild(x);
    box.appendChild(pill);
  });
  const inp=document.createElement('input');
  inp.type='time'; inp.className='acct-new-time'; inp.value='12:00';
  inp.style.width='auto'; inp.style.padding='4px 8px';
  const add=document.createElement('button');
  add.type='button'; add.className='btn-secondary'; add.style.padding='6px 10px';
  add.textContent='+';
  add.onclick=function(){ addAcctTime(add); };
  box.appendChild(inp); box.appendChild(add);
  if(!(accountTimes[id]&&accountTimes[id].length)){
    const eh=document.createElement('span');
    eh.className='hint';
    eh.setAttribute('data-testid','custom-empty-hint');
    eh.textContent=labelSchedCustomEmpty;
    row.appendChild(eh);
  } else {
    const old=row.querySelector('[data-testid="custom-empty-hint"]');
    if(old) old.remove();
  }
}
function removeAcctTime(btn, tm){
  const card=btn.closest('[data-testid="account-schedule"]');
  if(!card) return;
  const id=cardId(card);
  accountTimes[id]=(accountTimes[id]||[]).filter(x=>x!==tm);
  const row=btn.closest('.sched-row');
  renderMiniTimes(card, row);
}
function addAcctTime(btn){
  const card=btn.closest('[data-testid="account-schedule"]');
  if(!card) return;
  const id=cardId(card);
  const inp=card.querySelector('.acct-new-time');
  if(!inp || !inp.value) return;
  const n=inp.value.slice(0,5);
  if(!accountTimes[id]) accountTimes[id]=[];
  if(accountTimes[id].indexOf(n)<0) accountTimes[id].push(n);
  accountTimes[id].sort();
  renderMiniTimes(card, btn.closest('.sched-row'));
}
function selectedAccounts(){
  return Array.from(document.querySelectorAll('input.acct:checked')).map(c=>c.getAttribute('data-id'));
}
function collectAccountTimes(){
  const out={};
  selectedAccounts().forEach(function(id){
    if(accountSched[id]==='custom' && accountTimes[id] && accountTimes[id].length){
      out[id]=accountTimes[id].slice();
    }
  });
  return out;
}
const KEY_STORAGE='codex-selective-ping:management-key';
function restoreKeyFromSession(){
  const el=document.getElementById('management-key');
  if(!el || el.value.trim()) return;
  try{
    const v=sessionStorage.getItem(KEY_STORAGE);
    if(v) el.value=v;
  }catch(e){}
}
function persistKeyIfPresent(){
  const el=document.getElementById('management-key');
  const v=el ? el.value.trim() : '';
  if(!v) return;
  try{ sessionStorage.setItem(KEY_STORAGE, v); }catch(e){}
}
function key(){
  restoreKeyFromSession();
  const v=document.getElementById('management-key').value.trim();
  if(v) persistKeyIfPresent();
  return v;
}
async function saveCfg(){
  const o=document.getElementById('result'); const k=key();
  if(!k){ o.textContent=msgNeedKey; return; }
  const account_times=collectAccountTimes();
  const body={schedule_enabled:document.getElementById('schedule_enabled').checked, timezone:document.getElementById('tz').value.trim(), times:times, accounts:selectedAccounts(), account_times:account_times};
  o.textContent=msgSaving;
  try{
    const r=await fetch('/v0/management/plugins/codex-selective-ping/config',{method:'PATCH',headers:{'Authorization':'Bearer '+k,'Content-Type':'application/json'},body:JSON.stringify(body)});
    o.textContent=await r.text();
    if(r.ok){ setTimeout(()=>location.reload(),800);}
  }catch(e){ o.textContent=String(e); }
}
function setRunButtonsDisabled(on){
  document.querySelectorAll('[data-run-now]').forEach(function(b){
    if(on){
      if(!b.getAttribute('data-idle-label')){
        b.setAttribute('data-idle-label', (b.textContent||'').trim());
      }
      b.textContent = msgRunningLabel;
    } else {
      var idle = b.getAttribute('data-idle-label');
      if(idle){ b.textContent = idle; }
    }
    b.disabled = !!on;
  });
}
function setRunningUI(on){
  setRunButtonsDisabled(on);
  showRunningBadge(on);
}
function showRunningBadge(on){
  const el = document.getElementById('running-badge');
  if(el){ el.hidden = !on; }
  const banner = document.getElementById('running-banner');
  if(banner){ banner.hidden = !on; }
}
function enterRunningMode(msg){
  setRunButtonsDisabled(true);
  showRunningBadge(true);
  const o = document.getElementById('result');
  if(o && msg){ o.textContent = msg; }
  startPolling();
}
async function pollOnce(){
  const k = key();
  if(!k){ const o=document.getElementById('result'); if(o) o.textContent=msgNeedKey; return; }
  try{
    const r = await fetch(STATUS_URL,{headers:{'Authorization':'Bearer '+k}});
    const j = await r.json();
    if(j && j.running === false){
      if(pollTimer){ clearInterval(pollTimer); pollTimer=null; }
      location.reload();
      return;
    }
  }catch(e){
    const o=document.getElementById('result');
    if(o){ o.textContent=String(e); }
  }
}
function startPolling(){
  if(pollTimer) return;
  pollTimer = setInterval(pollOnce, POLL_MS);
  pollOnce();
}
async function runNow(){
  const o=document.getElementById('result'); const k=key();
  if(!k){ o.textContent=msgNeedKey; return; }
  if(pollTimer){ enterRunningMode(msgRunPolling); return; }
  o.textContent=msgStarting;
  try{
    const r=await fetch(RUN_URL,{method:'POST',headers:{'Authorization':'Bearer '+k}});
    const text=await r.text();
    o.textContent=text;
    if(r.ok || r.status===409){
      enterRunningMode(r.status===409 ? msgRunAlready : msgRunPolling);
    }
  }catch(e){ o.textContent=String(e); }
}
(function(){
  restoreKeyFromSession();
  const root = document.body;
  if(root && root.getAttribute('data-running')==='true'){
    enterRunningMode(msgRunPolling);
  }
})();
renderTimes();
</script>
</body></html>`,
		html.EscapeString(string(lang)),
		html.EscapeString(t("title")),
		runningAttr,
		html.EscapeString(t("subtitle")),
		langActive(lang, LangZhHant), html.EscapeString(t("lang_zh")),
		langActive(lang, LangEn), html.EscapeString(t("lang_en")),
		langActive(lang, LangJa), html.EscapeString(t("lang_ja")),
		statusPillClass(st.Enabled),
		html.EscapeString(enabled), html.EscapeString(st.Version),
		runningBadgeHidden, html.EscapeString(t("running_label")),
		runningBadgeHidden, html.EscapeString(t("running_label")), html.EscapeString(t("run_polling")),
		html.EscapeString(t("rail_now")),
		html.EscapeString(t("next_run")), html.EscapeString(railNext),
		html.EscapeString(t("rail_whitelist")), selectedN, len(st.Accounts),
		html.EscapeString(t("rail_model")), html.EscapeString(st.Model),
		html.EscapeString(t("timezone")), html.EscapeString(st.Timezone),
		html.EscapeString(t("rail_key")),
		html.EscapeString(t("key_placeholder")),
		html.EscapeString(t("save")),
		runNowAttrs,
		html.EscapeString(runButtonLabel),
		html.EscapeString(t("refresh")),
		html.EscapeString(t("actions_hint")),
		html.EscapeString(t("rhythm_title")),
		html.EscapeString(t("rhythm_sub")),
		timeline.String(),
		customNextNote,
		html.EscapeString(t("timezone")),
		html.EscapeString(st.Timezone),
		html.EscapeString(t("add_time")),
		html.EscapeString(t("add")),
		enabledChecked,
		html.EscapeString(t("enable")),
		html.EscapeString(t("how_title")),
		html.EscapeString(t("how_body")),
		html.EscapeString(t("how_global_off")), html.EscapeString(t("how_global_off_v")),
		html.EscapeString(t("how_manual")), html.EscapeString(t("how_manual_v")),
		html.EscapeString(t("how_quota_removed")), html.EscapeString(t("how_quota_removed_v")),
		html.EscapeString(t("status")), principlesChipClass(st.Enabled), html.EscapeString(enabled),
		accountsFilledClass,
		html.EscapeString(t("accounts_who_title")),
		html.EscapeString(t("accounts_times_sub")),
		html.EscapeString(fmt.Sprintf(t("filter_all"), len(st.Accounts))),
		html.EscapeString(fmt.Sprintf(t("filter_selected"), selectedN)),
		html.EscapeString(fmt.Sprintf(t("filter_custom"), customN)),
		html.EscapeString(t("select_all")),
		html.EscapeString(t("clear")),
		html.EscapeString(banner),
		rows,
		accountsEmptyClass,
		html.EscapeString(t("accounts_none_title")),
		html.EscapeString(t("accounts_empty_title")),
		html.EscapeString(t("accounts_none_body")),
		html.EscapeString(t("accounts_none_cta")),
		lastFilledClass,
		html.EscapeString(t("run_history_title")),
		html.EscapeString(t("run_history_sub")),
		lastBlock,
		lastEmptyClass,
		html.EscapeString(t("run_history_title")),
		html.EscapeString(t("last_run_empty_title")),
		html.EscapeString(t("last_run_empty_body")),
		runNowAttrs,
		html.EscapeString(runButtonLabel),
		string(timesJSON),
		string(accountTimesJSON),
		nextSlot,
		t("slot_next_inherit"),
		t("slot_past"),
		t("sched_effective"),
		t("sched_not_selected"),
		t("sched_inherit"),
		t("sched_custom"),
		t("sched_custom_empty"),
		t("hist_expand"),
		t("hist_collapse"),
		string(lang),
		needKey,
		saving,
		starting,
		runPolling,
		runAlready,
		t("running_label"),
	)
}

func langActive(current, target Lang) string {
	if current == target {
		return "active"
	}
	return ""
}

func statusLabel(lang Lang, status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success", "ok":
		return T(lang, "status_success")
	case "limited":
		return T(lang, "status_limited")
	case "failed", "error":
		return T(lang, "status_failed")
	case "skipped":
		return T(lang, "status_skipped")
	default:
		return status
	}
}

func preferID(a runstate.AccountView) string {
	if a.AuthIndex != "" {
		return a.AuthIndex
	}
	if a.Email != "" {
		return a.Email
	}
	return a.Name
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func statusPillClass(enabled bool) string {
	if enabled {
		return ""
	}
	return "neutral"
}

func principlesChipClass(enabled bool) string {
	if enabled {
		return "ok"
	}
	return ""
}

func rhythmSlot(tm, nextSlot, nowHHMM string) (class, captionKey string) {
	if nextSlot != "" && tm == nextSlot {
		return "slot next", "slot_next"
	}
	if nowHHMM != "" && tm < nowHHMM {
		return "slot", "slot_past"
	}
	return "slot", ""
}

func statusChipClass(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success", "ok", "available":
		return "ok"
	case "limited", "warn", "warning":
		return "warn"
	case "failed", "error", "unavailable", "bad":
		return "bad"
	default:
		return ""
	}
}
