package management

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"cpa-plugin-codex-selective-ping/internal/hostapi"
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
	// still highlight matching HH:MM even if not exact list membership after edits
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
	var timeline strings.Builder
	for _, tm := range st.Times {
		slotClass := "slot"
		caption := t("slot_past")
		if nextSlot != "" && tm == nextSlot {
			slotClass = "slot next"
			caption = t("slot_next")
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
	var rows strings.Builder
	for _, a := range st.Accounts {
		checked := ""
		cardClass := "acct"
		if a.Selected {
			checked = " checked"
			cardClass = "acct selected"
		}
		statusChip := statusChipClass(a.Status)
		meta := html.EscapeString(a.Email)
		if meta == "" {
			meta = html.EscapeString(t("auth_index_label"))
		}
		meta = meta + " · " + html.EscapeString(a.AuthIndex)
		fmt.Fprintf(&rows, `<div class="%s" data-auth-index="%s" data-name="%s" data-selected="%t" data-abnormal="%t">
<input type="checkbox" class="acct" data-id="%s"%s onchange="syncAcctCard(this)">
<div><div class="name">%s</div><div class="meta">%s</div></div>
<div class="hide-sm"><span class="chip" data-col="plan">%s</span></div>
<div class="hide-sm"><div class="quota-label" data-i18n="col_5h">%s</div><div class="quota" data-col="five_hour">%s</div></div>
<div class="hide-sm"><div class="quota-label" data-i18n="col_weekly">%s</div><div class="quota" data-col="weekly">%s</div></div>
<span class="chip %s" data-col="status">%s</span>
</div>`,
			cardClass,
			html.EscapeString(a.AuthIndex), html.EscapeString(a.Name), a.Selected, accountAbnormal(a),
			html.EscapeString(preferID(a)), checked,
			html.EscapeString(a.Name), meta,
			dash(a.Plan),
			html.EscapeString(t("col_5h")), formatWindow(a.FiveHour, lang),
			html.EscapeString(t("col_weekly")), formatWindow(a.Weekly, lang),
			statusChip, html.EscapeString(orDash(a.Status)),
		)
	}
	accountsEmptyClass := ""
	if selectedN > 0 {
		accountsEmptyClass = " hidden"
	}
	lastFilledClass := " hidden"
	lastEmptyClass := ""
	lastBlock := ""
	if st.LastRun != nil {
		lastFilledClass = ""
		lastEmptyClass = " hidden"
		lr := st.LastRun
		var lrRows strings.Builder
		for _, a := range lr.Accounts {
			meta := fmt.Sprintf("HTTP %d", a.HTTPStatus)
			if a.Error != "" {
				meta = html.EscapeString(a.Error)
				if a.HTTPStatus > 0 {
					meta = fmt.Sprintf("%d · %s", a.HTTPStatus, html.EscapeString(a.Error))
				}
			} else if a.HTTPStatus == 0 {
				meta = "—"
			}
			fmt.Fprintf(&lrRows, `<div class="run-item"><div><div class="name" style="font-weight:750">%s</div><div class="meta">%s</div></div><span class="chip %s">%s</span></div>`,
				html.EscapeString(a.Name), meta, statusChipClass(a.Status), html.EscapeString(statusLabel(lang, a.Status)))
		}
		big := fmt.Sprintf("%d %s", lr.Succeeded, t("chip_ok"))
		metaLine := fmt.Sprintf("%s · %s<br/>%s %d · %s %d", html.EscapeString(lr.Mode), html.EscapeString(lr.At.Format("15:04")), html.EscapeString(t("chip_limited")), lr.Limited, html.EscapeString(t("chip_skipped")), lr.Skipped)
		lastBlock = fmt.Sprintf(`<div class="run"><div class="run-summary"><div class="label" data-i18n="last_run_receipt_label">%s</div><div class="big">%s</div><div class="meta">%s</div></div><div class="run-list">%s</div></div>`,
			html.EscapeString(t("last_run_receipt_label")), html.EscapeString(big), metaLine, lrRows.String())
		if lr.Message != "" {
			lastBlock += `<p class="hint">` + html.EscapeString(lr.Message) + `</p>`
		}
	}
	timesJSON, _ := json.Marshal(st.Times)
	banner := fmt.Sprintf(t("banner_selected"), selectedN, len(st.Accounts))
	if len(st.AccountsConfig) == 0 {
		banner = t("banner_empty")
	}

	needKey := t("need_key")
	saving := t("saving")
	starting := t("starting")

	return fmt.Sprintf(`<!doctype html>
<html lang="%s">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>%s</title>
<style>
:root,:root[data-theme="light"]{--bg:#f4f6f8;--card:#fff;--text:#1f2937;--muted:#6b7280;--line:#e5e7eb;--brand:#2563eb;--chip:#eff6ff;--chip-text:#1d4ed8;--stat:#f9fafb;--pre:#f6f6f6;--btn-sec-bg:#e5e7eb;--btn-sec-text:#111;--banner-bg:#fff7ed;--banner-line:#fed7aa;--banner-text:#9a3412;--header-bg:#111827}
:root[data-theme="dark"]{--bg:#0b1220;--card:#111827;--text:#e5e7eb;--muted:#9ca3af;--line:#1f2937;--brand:#3b82f6;--chip:#1e3a8a;--chip-text:#bfdbfe;--stat:#0f172a;--pre:#0f172a;--btn-sec-bg:#1f2937;--btn-sec-text:#e5e7eb;--banner-bg:#451a03;--banner-line:#9a3412;--banner-text:#fed7aa;--header-bg:#020617}
*{box-sizing:border-box}body{margin:0;font-family:ui-sans-serif,system-ui,sans-serif;background:var(--bg);color:var(--text)}
header{background:var(--header-bg);color:#fff;padding:16px 24px;display:flex;justify-content:space-between;align-items:flex-start;gap:12px;flex-wrap:wrap}
header .titles h1{margin:0;font-size:18px}header .titles p{margin:4px 0 0;font-size:12px;opacity:.75}
.lang-switch{display:flex;gap:6px;flex-wrap:wrap}.lang-switch a{color:#fff;text-decoration:none;font-size:12px;padding:4px 8px;border-radius:999px;border:1px solid rgba(255,255,255,.35);opacity:.85}
.lang-switch a.active,.lang-switch a:hover{background:rgba(255,255,255,.15);opacity:1}
main{max-width:1100px;margin:20px auto;padding:0 16px 40px;display:grid;gap:16px}
.card{background:var(--card);border:1px solid var(--line);border-radius:12px;padding:16px 18px}
.card h2{margin:0 0 12px;font-size:15px}.grid{display:grid;grid-template-columns:repeat(4,1fr);gap:10px}
@media(max-width:800px){.grid{grid-template-columns:repeat(2,1fr)}}
.stat{background:var(--stat);border:1px solid var(--line);border-radius:10px;padding:10px 12px}
.stat .k{font-size:11px;color:var(--muted)}.stat .v{font-size:14px;font-weight:600;margin-top:4px}
.row{display:flex;gap:8px;flex-wrap:wrap;align-items:center}label{font-size:13px;color:var(--muted)}
input[type=text],input[type=password],input[type=time]{border:1px solid var(--line);border-radius:8px;padding:8px 10px;font-size:13px}
input[type=password]{min-width:220px}button{border:0;border-radius:8px;padding:8px 12px;font-size:13px;cursor:pointer}
.btn{background:var(--brand);color:#fff}.btn.secondary{background:var(--btn-sec-bg);color:var(--btn-sec-text)}
.hint{font-size:12px;color:var(--muted);margin-top:8px}
.tag{display:inline-block;padding:2px 8px;border-radius:999px;background:var(--chip);color:var(--chip-text);font-size:11px}
table{width:100%%;border-collapse:collapse;font-size:13px}th,td{padding:10px 8px;border-bottom:1px solid var(--line);text-align:left;vertical-align:top}
th{font-size:11px;color:var(--muted)}.quota{font-variant-numeric:tabular-nums;white-space:nowrap}.quota small{display:block;color:var(--muted);font-size:11px}
.banner{background:var(--banner-bg);border:1px solid var(--banner-line);color:var(--banner-text);border-radius:10px;padding:10px 12px;font-size:13px}
pre{white-space:pre-wrap;background:var(--pre);padding:12px;border-radius:6px}
</style>
</head>
<body>
<div class="shell">
<aside class="rail">
  <p class="brand">Selective<br/><em>Ping</em></p>
  <p class="lede" data-i18n="subtitle">%s</p>
  <nav class="lang-switch" aria-label="language">
    <a href="?lang=zh-Hant" class="%s" data-lang="zh-Hant">%s</a>
    <a href="?lang=en" class="%s" data-lang="en">%s</a>
    <a href="?lang=ja" class="%s" data-lang="ja">%s</a>
  </nav>
  <div class="status-pill"><i></i> %s · v%s</div>
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
      <button class="btn-secondary" type="button" onclick="runNow()" data-i18n="run_now">%s</button>
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
<p class="sub" data-i18n="schedule_hint">%s</p>
<div class="timeline" id="times">%s</div>
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
<h2 data-i18n="principles_title">%s</h2>
<p class="sub" data-i18n="principles_body">%s</p>
<div class="metric"><span class="k" data-i18n="status">%s</span><span class="v"><span class="chip ok">%s</span></span></div>
<div class="metric"><span class="k" data-i18n="col_5h">%s</span><span class="v" data-i18n="principles_quota">%s</span></div>
<div class="metric"><span class="k" data-i18n="last_run">%s</span><span class="v" data-i18n="principles_persist">%s</span></div>
</section>
</section>
<section class="panel" id="sec-accounts-filled">
<div class="accounts-head">
  <div>
    <h2 data-i18n="accounts_who_title">%s</h2>
    <p class="sub" data-i18n="accounts_hint" style="margin:0">%s</p>
  </div>
  <div class="tabs">
    <button class="tab on" type="button" data-filter="all" onclick="filterAccounts('all')" data-i18n="filter_all">%s</button>
    <button class="tab" type="button" data-filter="selected" onclick="filterAccounts('selected')" data-i18n="filter_selected">%s</button>
    <button class="tab" type="button" data-filter="abnormal" onclick="filterAccounts('abnormal')" data-i18n="filter_abnormal">%s</button>
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
  <h2 data-i18n="accounts_empty_heading">%s</h2>
  <div class="empty">
    <strong data-i18n="accounts_empty_title">%s</strong>
    <p data-i18n="accounts_empty_body">%s</p>
    <button class="btn-primary" type="button" onclick="document.getElementById('sec-accounts-filled')?.scrollIntoView({behavior:'smooth'})" data-i18n="accounts_empty_cta">%s</button>
  </div>
</section>
<section class="panel%s" id="sec-last-filled">
<h2 data-i18n="last_run">%s</h2>
<p class="sub" data-i18n="last_run_sub">%s</p>
%s
</section>
<section class="panel%s" id="sec-last-empty">
<h2 data-i18n="last_run">%s</h2>
<div class="empty">
<strong data-i18n="last_run_empty_title">%s</strong>
<p data-i18n="last_run_empty_body">%s</p>
<button class="btn-secondary" type="button" onclick="runNow()" data-i18n="run_now">%s</button>
</div>
</section>
</div>
</div>
<script>
const initialTimes = %s;
const nextSlotHint = %q;
const slotCaptionNext = %q;
const slotCaptionPast = %q;
const pluginId = "codex-selective-ping";
const serverLang = %q;
const msgNeedKey = %q;
const msgSaving = %q;
const msgStarting = %q;
let times = Array.isArray(initialTimes) ? initialTimes.slice() : [];

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
  // zh-CN / ru / anything else → zh-Hant
  return 'zh-Hant';
}

(function alignCPALanguage(){
  try{
    const params = new URLSearchParams(location.search);
    if(params.has('lang')) return; // explicit override; do not write CPA localStorage
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
function renderTimes(){
  const el = document.getElementById('times');
  if(!el) return;
  el.innerHTML = '';
  times.forEach((tm)=>{
    const d = document.createElement('div');
    const isNext = nextSlotHint && tm === nextSlotHint;
    d.className = isNext ? 'slot next' : 'slot';
    d.setAttribute('data-time', tm);
    const x = document.createElement('button');
    x.type='button'; x.className='x'; x.setAttribute('aria-label','remove'); x.textContent='×';
    x.onclick=()=>removeTime(tm);
    const tEl = document.createElement('div'); tEl.className='t'; tEl.textContent=tm;
    const c = document.createElement('div'); c.className='caption'; c.textContent = isNext ? slotCaptionNext : slotCaptionPast;
    d.appendChild(x); d.appendChild(tEl); d.appendChild(c); el.appendChild(d);
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
function setAll(v){
  document.querySelectorAll('input.acct').forEach(c=>{ c.checked=v; syncAcctCard(c); });
  const empty=document.getElementById('sec-accounts-empty');
  if(empty){ empty.classList.toggle('hidden', !!v && document.querySelectorAll('input.acct:checked').length>0); }
}
function syncAcctCard(cb){
  const card=cb.closest('.acct[data-auth-index],div[data-auth-index]');
  if(!card) return;
  card.classList.toggle('selected', cb.checked);
  card.setAttribute('data-selected', cb.checked ? 'true' : 'false');
  const empty=document.getElementById('sec-accounts-empty');
  if(empty){
    const n=document.querySelectorAll('input.acct:checked').length;
    empty.classList.toggle('hidden', n>0);
  }
}
function filterAccounts(mode){
  document.querySelectorAll('.tabs .tab').forEach(t=>t.classList.toggle('on', t.getAttribute('data-filter')===mode));
  document.querySelectorAll('#account-grid > [data-auth-index]').forEach(card=>{
    const sel=card.getAttribute('data-selected')==='true';
    const abn=card.getAttribute('data-abnormal')==='true';
    let show=true;
    if(mode==='selected') show=sel;
    if(mode==='abnormal') show=abn;
    card.style.display = show ? '' : 'none';
  });
}
function selectedAccounts(){
  return Array.from(document.querySelectorAll('input.acct:checked')).map(c=>c.getAttribute('data-id'));
}
function key(){ return document.getElementById('management-key').value.trim(); }
async function saveCfg(){
  const o=document.getElementById('result'); const k=key();
  if(!k){ o.textContent=msgNeedKey; return; }
  const body={schedule_enabled:document.getElementById('schedule_enabled').checked, timezone:document.getElementById('tz').value.trim(), times:times, accounts:selectedAccounts()};
  o.textContent=msgSaving;
  try{
    const r=await fetch('/v0/management/plugins/codex-selective-ping/config',{method:'PATCH',headers:{'Authorization':'Bearer '+k,'Content-Type':'application/json'},body:JSON.stringify(body)});
    o.textContent=await r.text();
    if(r.ok){ enrichQuotaFromManagement(); setTimeout(()=>location.reload(),800);} 
  }catch(e){ o.textContent=String(e); }
}
async function runNow(){
  const o=document.getElementById('result'); const k=key();
  if(!k){ o.textContent=msgNeedKey; return; }
  o.textContent=msgStarting;
  try{
    const r=await fetch('/v0/management/plugins/codex-selective-ping/run',{method:'POST',headers:{'Authorization':'Bearer '+k}});
    o.textContent=await r.text();
    if(r.ok){ enrichQuotaFromManagement(); setTimeout(()=>location.reload(),1500);} 
  }catch(e){ o.textContent=String(e); }
}
function fmtQuotaWindow(w){
  if(!w) return '—';
  const parts=[];
  if(w.remaining!=null && w.remaining!=='') parts.push(('剩/left/残'.split('/')[0])+' '+w.remaining);
  // Keep labels minimal/numeric; server-rendered i18n already covers first paint.
  if(w.remaining!=null) parts.push(String(w.remaining));
  if(w.used!=null) parts.push('used '+w.used);
  let main = parts.length ? (w.remaining!=null ? String(Number(w.remaining).toPrecision(4)).replace(/\.?0+$/,'') : ('used '+w.used)) : '—';
  if(w.remaining!=null){
    const n=Number(w.remaining);
    main = (Number.isFinite(n)? n.toPrecision(4).replace(/\.?0+$/,'') : String(w.remaining));
  } else if(w.used!=null){
    const n=Number(w.used);
    main = 'used '+(Number.isFinite(n)? n.toPrecision(4).replace(/\.?0+$/,'') : String(w.used));
  }
  let reset='';
  if(w.resets_at){
    try{
      const d=new Date(w.resets_at);
      if(!isNaN(d.getTime())){
        const mm=String(d.getMonth()+1).padStart(2,'0');
        const dd=String(d.getDate()).padStart(2,'0');
        const hh=String(d.getHours()).padStart(2,'0');
        const mi=String(d.getMinutes()).padStart(2,'0');
        reset='<small>'+mm+'-'+dd+' '+hh+':'+mi+'</small>';
      }
    }catch(e){}
  }
  if(main==='—' && !reset) return '—';
  return main+reset;
}
function planFromAuthFile(entry){
  if(!entry||typeof entry!=='object') return '';
  if(entry.plan) return String(entry.plan);
  if(entry.plan_type) return String(entry.plan_type);
  if(entry.planType) return String(entry.planType);
  const idt=entry.id_token;
  if(idt && typeof idt==='object'){
    return String(idt.plan_type||idt.planType||idt.chatgpt_plan_type||'');
  }
  return '';
}
function windowFromAuthFile(entry, which){
  if(!entry||typeof entry!=='object') return null;
  // Prefer explicit five_hour/weekly shapes if CPA ever exposes them.
  const direct=entry[which]||entry[which=== 'five_hour'?'fiveHour':'weeklyLimit'];
  if(direct && typeof direct==='object'){
    const rem=direct.remaining??direct.remaining_fraction??direct.remainingFraction??direct.remaining_percent??direct.remainingPercent;
    const used=direct.used??direct.used_percent??direct.usedPercent;
    const resets=direct.resets_at??direct.reset_at??direct.resetsAt??direct.resetAt;
    if(rem!=null||used!=null||resets!=null){
      const w={}; if(rem!=null) w.remaining=Number(rem); if(used!=null) w.used=Number(used); if(resets!=null) w.resets_at=resets; return w;
    }
  }
  return null;
}
function windowFromWham(usage, which){
  if(!usage||typeof usage!=='object') return null;
  const rate=usage.rate_limit||usage.rateLimit||{};
  let win=null;
  if(which==='five_hour'){
    win=rate.primary_window||rate.primaryWindow||null;
    if(win && Number(win.limit_window_seconds||win.limitWindowSeconds||0) && Number(win.limit_window_seconds||win.limitWindowSeconds)!==18000){
      // keep; duration matcher below can still use it
    }
  } else {
    win=rate.secondary_window||rate.secondaryWindow||null;
  }
  // Duration-based fallback across both windows.
  const cands=[rate.primary_window||rate.primaryWindow, rate.secondary_window||rate.secondaryWindow].filter(Boolean);
  if(which==='five_hour'){
    const byDur=cands.find(w=>Number(w.limit_window_seconds||w.limitWindowSeconds||0)===18000);
    if(byDur) win=byDur;
  } else {
    const byDur=cands.find(w=>{const s=Number(w.limit_window_seconds||w.limitWindowSeconds||0); return s===604800 || (s>=2419200&&s<=2678400);});
    if(byDur) win=byDur;
  }
  if(!win) return null;
  const used=win.used_percent??win.usedPercent;
  const rem = used!=null ? (100-Number(used)) : (win.remaining_percent??win.remainingPercent??win.remaining);
  const resets=win.reset_at??win.resetAt??win.resets_at??win.resetsAt;
  if(rem==null && used==null && !resets) return null;
  const out={};
  if(rem!=null) out.remaining=Number(rem);
  if(used!=null) out.used=Number(used);
  if(resets!=null){
    // unix seconds → ISO
    const n=Number(resets);
    out.resets_at = (Number.isFinite(n) && n>1000000000) ? new Date(n*1000).toISOString() : String(resets);
  }
  return out;
}
function applyQuotaToRow(row, plan, five, weekly){
  if(plan){ const el=row.querySelector('[data-col="plan"]'); if(el) el.textContent=plan; }
  if(five){ const el=row.querySelector('[data-col="five_hour"]'); if(el) el.innerHTML=fmtQuotaWindow(five); }
  if(weekly){ const el=row.querySelector('[data-col="weekly"]'); if(el) el.innerHTML=fmtQuotaWindow(weekly); }
}
async function enrichQuotaFromManagement(){
  const k=key();
  if(!k) return;
  const o=document.getElementById('result');
  try{
    const r=await fetch('/v0/management/auth-files',{headers:{'Authorization':'Bearer '+k,'Accept':'application/json'}});
    if(!r.ok){ if(o) o.textContent='auth-files HTTP '+r.status; return; }
    const data=await r.json();
    const files=Array.isArray(data)?data:(data.files||data.items||[]);
    const byIndex={}; const byName={};
    files.forEach(f=>{
      if(!f) return;
      const idx=String(f.auth_index||f.authIndex||'');
      const name=String(f.name||'');
      if(idx) byIndex[idx]=f;
      if(name) byName[name]=f;
    });
    const rows=Array.from(document.querySelectorAll('[data-auth-index]'));
    for(const row of rows){
      const idx=row.getAttribute('data-auth-index')||'';
      const name=row.getAttribute('data-name')||'';
      const entry=byIndex[idx]||byName[name];
      if(!entry) continue;
      const plan=planFromAuthFile(entry);
      let five=windowFromAuthFile(entry,'five_hour');
      let weekly=windowFromAuthFile(entry,'weekly');
      // Live 5h/weekly: same source CPA admin uses (api-call → wham/usage).
      try{
        const accountId=(entry.id_token&& (entry.id_token.chatgpt_account_id||entry.id_token.chatgptAccountId))||'';
        const header={'Authorization':'Bearer $TOKEN$','Content-Type':'application/json','Accept':'application/json'};
        if(accountId) header['Chatgpt-Account-Id']=accountId;
        const ur=await fetch('/v0/management/api-call',{
          method:'POST',
          headers:{'Authorization':'Bearer '+k,'Content-Type':'application/json','Accept':'application/json'},
          body:JSON.stringify({authIndex:idx, method:'GET', url:'https://chatgpt.com/backend-api/wham/usage', header:header})
        });
        if(ur.ok){
          const uj=await ur.json();
          const body=uj.body!=null?uj.body:(uj.Body!=null?uj.Body:uj);
          let usage=body;
          if(typeof usage==='string'){ try{usage=JSON.parse(usage);}catch(e){usage=null;} }
          if(usage && typeof usage==='object'){
            const p2=usage.plan_type||usage.planType||'';
            if(p2 && !plan) {/* fill below */}
            const f2=windowFromWham(usage,'five_hour');
            const w2=windowFromWham(usage,'weekly');
            if(f2) five=f2;
            if(w2) weekly=w2;
            applyQuotaToRow(row, plan||p2||'', five, weekly);
            continue;
          }
        }
      }catch(e){ /* keep auth-files plan only */ }
      applyQuotaToRow(row, plan, five, weekly);
    }
  }catch(e){ if(o) o.textContent=String(e); }
}
const keyInput=document.getElementById('management-key');
if(keyInput){
  keyInput.addEventListener('change', ()=>{ enrichQuotaFromManagement(); });
  keyInput.addEventListener('blur', ()=>{ enrichQuotaFromManagement(); });
}
renderTimes();
</script>
</body></html>`,
		html.EscapeString(string(lang)),
		html.EscapeString(t("title")),
		html.EscapeString(t("subtitle")),
		langActive(lang, LangZhHant), html.EscapeString(t("lang_zh")),
		langActive(lang, LangEn), html.EscapeString(t("lang_en")),
		langActive(lang, LangJa), html.EscapeString(t("lang_ja")),
		html.EscapeString(enabled), html.EscapeString(st.Version),
		html.EscapeString(t("rail_now")),
		html.EscapeString(t("next_run")), html.EscapeString(railNext),
		html.EscapeString(t("rail_whitelist")), selectedN, len(st.Accounts),
		html.EscapeString(t("rail_model")), html.EscapeString(st.Model),
		html.EscapeString(t("timezone")), html.EscapeString(st.Timezone),
		html.EscapeString(t("rail_key")),
		html.EscapeString(t("key_placeholder")),
		html.EscapeString(t("save")),
		html.EscapeString(t("run_now")),
		html.EscapeString(t("refresh")),
		html.EscapeString(t("actions_hint")),
		html.EscapeString(t("rhythm_title")),
		html.EscapeString(t("schedule_hint")),
		timeline.String(),
		html.EscapeString(t("timezone")),
		html.EscapeString(st.Timezone),
		html.EscapeString(t("add_time")),
		html.EscapeString(t("add")),
		enabledChecked,
		html.EscapeString(t("enable")),
		html.EscapeString(t("principles_title")),
		html.EscapeString(t("principles_body")),
		html.EscapeString(t("status")), html.EscapeString(enabled),
		html.EscapeString(t("col_5h")), html.EscapeString(t("principles_quota")),
		html.EscapeString(t("last_run")), html.EscapeString(t("principles_persist")),
		html.EscapeString(t("accounts_who_title")),
		html.EscapeString(t("accounts_hint")),
		html.EscapeString(fmt.Sprintf(t("filter_all"), len(st.Accounts))),
		html.EscapeString(fmt.Sprintf(t("filter_selected"), selectedN)),
		html.EscapeString(t("filter_abnormal")),
		html.EscapeString(t("select_all")),
		html.EscapeString(t("clear")),
		html.EscapeString(banner),
		rows.String(),
		accountsEmptyClass,
		html.EscapeString(t("accounts_empty_heading")),
		html.EscapeString(t("accounts_empty_title")),
		html.EscapeString(t("accounts_empty_body")),
		html.EscapeString(t("accounts_empty_cta")),
		lastFilledClass,
		html.EscapeString(t("last_run")),
		html.EscapeString(t("last_run_sub")),
		lastBlock,
		lastEmptyClass,
		html.EscapeString(t("last_run")),
		html.EscapeString(t("last_run_empty_title")),
		html.EscapeString(t("last_run_empty_body")),
		html.EscapeString(t("run_now")),
		string(timesJSON),
		nextSlot,
		t("slot_next"),
		t("slot_past"),
		string(lang),
		needKey,
		saving,
		starting,
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
	case "success":
		return T(lang, "status_success")
	case "limited":
		return T(lang, "status_limited")
	case "failed":
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

func dash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return html.EscapeString(s)
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func formatWindow(w *hostapi.QuotaWindow, lang Lang) string {
	if w == nil {
		return "—"
	}
	parts := []string{}
	if w.Remaining != nil {
		parts = append(parts, fmt.Sprintf("%s %.4g", T(lang, "quota_remain"), *w.Remaining))
	}
	if w.Used != nil {
		parts = append(parts, fmt.Sprintf("%s %.4g", T(lang, "quota_used"), *w.Used))
	}
	main := "—"
	if len(parts) > 0 {
		main = strings.Join(parts, " / ")
	}
	reset := ""
	if w.ResetsAt != nil && !w.ResetsAt.IsZero() {
		reset = `<small>` + html.EscapeString(T(lang, "quota_reset")) + ` ` + html.EscapeString(w.ResetsAt.Format("01-02 15:04")) + `</small>`
	}
	if main == "—" && reset == "" {
		return "—"
	}
	return main + reset
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

func accountAbnormal(a runstate.AccountView) bool {
	if a.Unavailable || a.Disabled {
		return true
	}
	s := strings.ToLower(strings.TrimSpace(a.Status))
	return s == "limited" || s == "failed" || s == "unavailable" || s == "error"
}
