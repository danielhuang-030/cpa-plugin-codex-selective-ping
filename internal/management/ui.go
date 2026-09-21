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
	running := t("no")
	if st.Running {
		running = t("yes")
	}
	enabled := t("enabled_off")
	if st.Enabled {
		enabled = t("enabled_on")
	}
	enabledChecked := ""
	if st.Enabled {
		enabledChecked = " checked"
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
		if a.Selected {
			checked = " checked"
		}
		fmt.Fprintf(&rows, `<tr data-auth-index="%s" data-name="%s">
<td><input type="checkbox" class="acct" data-id="%s"%s></td>
<td>%s<br><small>%s · %s: %s</small></td>
<td class="quota-plan" data-col="plan">%s</td>
<td class="quota" data-col="five_hour">%s</td>
<td class="quota" data-col="weekly">%s</td>
<td>%s</td>
</tr>`,
			html.EscapeString(a.AuthIndex), html.EscapeString(a.Name),
			html.EscapeString(preferID(a)), checked,
			html.EscapeString(a.Name), html.EscapeString(a.Email), html.EscapeString(t("auth_index_label")), html.EscapeString(a.AuthIndex),
			dash(a.Plan),
			formatWindow(a.FiveHour, lang),
			formatWindow(a.Weekly, lang),
			html.EscapeString(orDash(a.Status)),
		)
	}
	lastBlock := `<p class="hint" data-i18n="no_last_run">` + html.EscapeString(t("no_last_run")) + `</p>`
	if st.LastRun != nil {
		lr := st.LastRun
		var lrRows strings.Builder
		for _, a := range lr.Accounts {
			fmt.Fprintf(&lrRows, `<tr><td>%s</td><td>%s</td><td>%d</td><td>%s</td></tr>`,
				html.EscapeString(a.Name), html.EscapeString(statusLabel(lang, a.Status)), a.HTTPStatus, html.EscapeString(orDash(a.Error)))
		}
		lastBlock = fmt.Sprintf(`<div class="row" style="margin-bottom:8px">
<span class="tag">%s: %s</span>
<span class="tag">%s</span>
<span class="tag">%s %d</span>
<span class="tag">%s %d</span>
<span class="tag">%s %d</span>
<span class="tag">%s %d</span>
</div>
<table><thead><tr><th data-i18n="col_account">%s</th><th data-i18n="col_result">%s</th><th data-i18n="col_http">%s</th><th data-i18n="col_detail">%s</th></tr></thead><tbody>%s</tbody></table>`,
			html.EscapeString(t("chip_mode")), html.EscapeString(lr.Mode), html.EscapeString(lr.At.Format(time.RFC3339)),
			html.EscapeString(t("chip_ok")), lr.Succeeded,
			html.EscapeString(t("chip_limited")), lr.Limited,
			html.EscapeString(t("chip_failed")), lr.Failed,
			html.EscapeString(t("chip_skipped")), lr.Skipped,
			html.EscapeString(t("col_account")), html.EscapeString(t("col_result")),
			html.EscapeString(t("col_http")), html.EscapeString(t("col_detail")),
			lrRows.String())
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
</aside>
<div class="workspace">
<section class="card"><h2 data-i18n="overview">%s</h2>
<div class="grid">
  <div class="stat"><div class="k" data-i18n="status">%s</div><div class="v">%s</div></div>
  <div class="stat"><div class="k" data-i18n="version_model">%s</div><div class="v">%s · %s</div></div>
  <div class="stat"><div class="k" data-i18n="tz_times">%s</div><div class="v">%s · %s</div></div>
  <div class="stat"><div class="k" data-i18n="next_run">%s</div><div class="v">%s · <span data-i18n="running_label">%s</span>：%s</div></div>
</div></section>
<section class="card"><h2 data-i18n="schedule">%s</h2>
<div class="row" style="margin-bottom:10px">
  <label><input id="schedule_enabled" type="checkbox"%s/> <span data-i18n="enable">%s</span></label>
  <label data-i18n="timezone">%s</label><input id="tz" type="text" value="%s"/>
  <label data-i18n="add_time">%s</label><input id="new-time" type="time" value="21:00"/>
  <button class="btn secondary" type="button" onclick="addTime()" data-i18n="add">%s</button>
</div>
<div class="row" id="times"></div>
<p class="hint" data-i18n="schedule_hint">%s</p>
</section>
<section class="card"><h2 data-i18n="accounts">%s</h2>
<div class="banner" id="banner">%s</div>
<div class="row" style="margin:12px 0">
  <button class="btn secondary" type="button" onclick="setAll(true)" data-i18n="select_all">%s</button>
  <button class="btn secondary" type="button" onclick="setAll(false)" data-i18n="clear">%s</button>
</div>
<table>
<thead><tr><th></th><th data-i18n="col_account">%s</th><th data-i18n="col_plan">%s</th><th data-i18n="col_5h">%s</th><th data-i18n="col_weekly">%s</th><th data-i18n="col_status">%s</th></tr></thead>
<tbody>%s</tbody>
</table>
<p class="hint" data-i18n="accounts_hint">%s</p>
</section>
<section class="card"><h2 data-i18n="actions">%s</h2>
<div class="row">
  <input id="management-key" type="password" autocomplete="off" data-i18n-placeholder="key_placeholder" placeholder="%s"/>
  <button class="btn" type="button" onclick="saveCfg()" data-i18n="save">%s</button>
  <button class="btn secondary" type="button" onclick="runNow()" data-i18n="run_now">%s</button>
  <button class="btn secondary" type="button" onclick="location.reload()" data-i18n="refresh">%s</button>
</div>
<p class="hint" data-i18n="actions_hint">%s</p>
<pre id="result"></pre>
</section>
<section class="card"><h2 data-i18n="last_run">%s</h2>%s</section>
</div>
</div>
<script>
const initialTimes = %s;
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
  el.innerHTML = '';
  times.forEach((t,i)=>{
    const s = document.createElement('span');
    s.className = 'tag';
    s.textContent = t + ' ';
    const x = document.createElement('button');
    x.type='button'; x.textContent='×'; x.className='btn secondary';
    x.onclick=()=>{ times.splice(i,1); renderTimes(); };
    s.appendChild(x); el.appendChild(s);
  });
}
function addTime(){
  const v = document.getElementById('new-time').value;
  if(!v) return;
  const n = v.slice(0,5);
  if(!times.includes(n)) times.push(n);
  times.sort(); renderTimes();
}
function setAll(v){ document.querySelectorAll('input.acct').forEach(c=>c.checked=v); }
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
    const rows=Array.from(document.querySelectorAll('tr[data-auth-index]'));
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
		html.EscapeString(t("overview")),
		html.EscapeString(t("status")), html.EscapeString(enabled),
		html.EscapeString(t("version_model")),
		html.EscapeString(st.Version), html.EscapeString(st.Model),
		html.EscapeString(t("tz_times")),
		html.EscapeString(st.Timezone), html.EscapeString(strings.Join(st.Times, " / ")),
		html.EscapeString(t("next_run")),
		html.EscapeString(next), html.EscapeString(t("running_label")), html.EscapeString(running),
		html.EscapeString(t("schedule")),
		enabledChecked,
		html.EscapeString(t("enable")),
		html.EscapeString(t("timezone")),
		html.EscapeString(st.Timezone),
		html.EscapeString(t("add_time")),
		html.EscapeString(t("add")),
		html.EscapeString(t("schedule_hint")),
		html.EscapeString(t("accounts")),
		html.EscapeString(banner),
		html.EscapeString(t("select_all")),
		html.EscapeString(t("clear")),
		html.EscapeString(t("col_account")),
		html.EscapeString(t("col_plan")),
		html.EscapeString(t("col_5h")),
		html.EscapeString(t("col_weekly")),
		html.EscapeString(t("col_status")),
		rows.String(),
		html.EscapeString(t("accounts_hint")),
		html.EscapeString(t("actions")),
		html.EscapeString(t("key_placeholder")),
		html.EscapeString(t("save")),
		html.EscapeString(t("run_now")),
		html.EscapeString(t("refresh")),
		html.EscapeString(t("actions_hint")),
		html.EscapeString(t("last_run")),
		lastBlock,
		string(timesJSON),
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
