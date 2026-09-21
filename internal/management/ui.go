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

func RenderStatusPage(st StatusResponse) string {
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
	running := "否"
	if st.Running {
		running = "是"
	}
	enabled := "停用"
	if st.Enabled {
		enabled = "已啟用"
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
		fmt.Fprintf(&rows, `<tr>
<td><input type="checkbox" class="acct" data-id="%s"%s></td>
<td>%s<br><small>%s · auth_index: %s</small></td>
<td>%s</td>
<td class="quota">%s</td>
<td class="quota">%s</td>
<td>%s</td>
</tr>`,
			html.EscapeString(preferID(a)), checked,
			html.EscapeString(a.Name), html.EscapeString(a.Email), html.EscapeString(a.AuthIndex),
			dash(a.Plan),
			formatWindow(a.FiveHour),
			formatWindow(a.Weekly),
			html.EscapeString(orDash(a.Status)),
		)
	}
	lastBlock := `<p class="hint">尚無執行紀錄</p>`
	if st.LastRun != nil {
		lr := st.LastRun
		var lrRows strings.Builder
		for _, a := range lr.Accounts {
			fmt.Fprintf(&lrRows, `<tr><td>%s</td><td>%s</td><td>%d</td><td>%s</td></tr>`,
				html.EscapeString(a.Name), html.EscapeString(a.Status), a.HTTPStatus, html.EscapeString(orDash(a.Error)))
		}
		lastBlock = fmt.Sprintf(`<div class="row" style="margin-bottom:8px">
<span class="tag">mode: %s</span>
<span class="tag">%s</span>
<span class="tag">ok %d</span>
<span class="tag">limited %d</span>
<span class="tag">failed %d</span>
<span class="tag">skipped %d</span>
</div>
<table><thead><tr><th>帳號</th><th>結果</th><th>HTTP</th><th>說明</th></tr></thead><tbody>%s</tbody></table>`,
			html.EscapeString(lr.Mode), html.EscapeString(lr.At.Format(time.RFC3339)),
			lr.Succeeded, lr.Limited, lr.Failed, lr.Skipped, lrRows.String())
		if lr.Message != "" {
			lastBlock += `<p class="hint">` + html.EscapeString(lr.Message) + `</p>`
		}
	}
	timesJSON, _ := json.Marshal(st.Times)
	banner := fmt.Sprintf("目前已勾選 %d / %d 個帳號。未勾選的不會被排程或「立刻執行」打到。", selectedN, len(st.Accounts))
	if len(st.AccountsConfig) == 0 {
		banner = "帳號白名單為空：排程與立刻執行都不會 ping 任何人。"
	}
	return fmt.Sprintf(`<!doctype html>
<html lang="zh-Hant">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>Codex Selective Ping</title>
<style>
:root{--bg:#f4f6f8;--card:#fff;--text:#1f2937;--muted:#6b7280;--line:#e5e7eb;--brand:#2563eb;--chip:#eff6ff;--chip-text:#1d4ed8}
*{box-sizing:border-box}body{margin:0;font-family:ui-sans-serif,system-ui,sans-serif;background:var(--bg);color:var(--text)}
header{background:#111827;color:#fff;padding:16px 24px}header h1{margin:0;font-size:18px}header p{margin:4px 0 0;font-size:12px;opacity:.75}
main{max-width:1100px;margin:20px auto;padding:0 16px 40px;display:grid;gap:16px}
.card{background:var(--card);border:1px solid var(--line);border-radius:12px;padding:16px 18px}
.card h2{margin:0 0 12px;font-size:15px}.grid{display:grid;grid-template-columns:repeat(4,1fr);gap:10px}
@media(max-width:800px){.grid{grid-template-columns:repeat(2,1fr)}}
.stat{background:#f9fafb;border:1px solid var(--line);border-radius:10px;padding:10px 12px}
.stat .k{font-size:11px;color:var(--muted)}.stat .v{font-size:14px;font-weight:600;margin-top:4px}
.row{display:flex;gap:8px;flex-wrap:wrap;align-items:center}label{font-size:13px;color:var(--muted)}
input[type=text],input[type=password],input[type=time]{border:1px solid var(--line);border-radius:8px;padding:8px 10px;font-size:13px}
input[type=password]{min-width:220px}button{border:0;border-radius:8px;padding:8px 12px;font-size:13px;cursor:pointer}
.btn{background:var(--brand);color:#fff}.btn.secondary{background:#e5e7eb;color:#111}
.hint{font-size:12px;color:var(--muted);margin-top:8px}
.tag{display:inline-block;padding:2px 8px;border-radius:999px;background:var(--chip);color:var(--chip-text);font-size:11px}
table{width:100%%;border-collapse:collapse;font-size:13px}th,td{padding:10px 8px;border-bottom:1px solid var(--line);text-align:left;vertical-align:top}
th{font-size:11px;color:var(--muted)}.quota{font-variant-numeric:tabular-nums;white-space:nowrap}.quota small{display:block;color:var(--muted);font-size:11px}
.banner{background:#fff7ed;border:1px solid #fed7aa;color:#9a3412;border-radius:10px;padding:10px 12px;font-size:13px}
pre{white-space:pre-wrap;background:#f6f6f6;padding:12px;border-radius:6px}
</style>
</head>
<body>
<header>
  <h1>Codex Selective Ping</h1>
  <p>獨立 CPA 插件 · 指定帳號排程 ping</p>
</header>
<main>
<section class="card"><h2>概況</h2>
<div class="grid">
  <div class="stat"><div class="k">狀態</div><div class="v">%s</div></div>
  <div class="stat"><div class="k">版本 / 模型</div><div class="v">%s · %s</div></div>
  <div class="stat"><div class="k">時區 / 時段</div><div class="v">%s · %s</div></div>
  <div class="stat"><div class="k">下次執行</div><div class="v">%s · 執行中：%s</div></div>
</div></section>
<section class="card"><h2>排程</h2>
<div class="row" style="margin-bottom:10px">
  <label>時區</label><input id="tz" type="text" value="%s"/>
  <label>新增時段</label><input id="new-time" type="time" value="21:00"/>
  <button class="btn secondary" type="button" onclick="addTime()">加入</button>
</div>
<div class="row" id="times"></div>
<p class="hint">啟動時不會立刻全 ping；到點才跑。空帳號清單 = 誰都不 ping。</p>
</section>
<section class="card"><h2>帳號</h2>
<div class="banner" id="banner">%s</div>
<div class="row" style="margin:12px 0">
  <button class="btn secondary" type="button" onclick="setAll(true)">全選</button>
  <button class="btn secondary" type="button" onclick="setAll(false)">清除</button>
</div>
<table>
<thead><tr><th></th><th>帳號</th><th>Plan</th><th>5h</th><th>週限</th><th>狀態</th></tr></thead>
<tbody>%s</tbody>
</table>
<p class="hint">5h／週限來自 CPA 既有資料；沒有就顯示 —，不會自己推估。額度欄只供參考，不決定是否 ping。</p>
</section>
<section class="card"><h2>動作</h2>
<div class="row">
  <input id="management-key" type="password" autocomplete="off" placeholder="Management Key（僅當次輸入，不持久化）"/>
  <button class="btn" type="button" onclick="saveCfg()">儲存設定</button>
  <button class="btn secondary" type="button" onclick="runNow()">立刻執行</button>
  <button class="btn secondary" type="button" onclick="location.reload()">重新整理</button>
</div>
<p class="hint">儲存：PATCH 宿主 plugins.configs。立刻執行：只 ping 目前已儲存白名單中的帳號（請先儲存勾選）。</p>
<pre id="result"></pre>
</section>
<section class="card"><h2>最近一次結果</h2>%s</section>
</main>
<script>
const initialTimes = %s;
const pluginId = "codex-selective-ping";
let times = Array.isArray(initialTimes) ? initialTimes.slice() : [];
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
  if(!k){ o.textContent='需要 Management Key'; return; }
  const body={enabled:true, timezone:document.getElementById('tz').value.trim(), times:times, accounts:selectedAccounts()};
  o.textContent='儲存中...';
  try{
    const r=await fetch('/v0/management/plugins/codex-selective-ping/config',{method:'PATCH',headers:{'Authorization':'Bearer '+k,'Content-Type':'application/json'},body:JSON.stringify(body)});
    o.textContent=await r.text();
    if(r.ok) setTimeout(()=>location.reload(),800);
  }catch(e){ o.textContent=String(e); }
}
async function runNow(){
  const o=document.getElementById('result'); const k=key();
  if(!k){ o.textContent='需要 Management Key'; return; }
  o.textContent='啟動中...';
  try{
    const r=await fetch('/v0/management/plugins/codex-selective-ping/run',{method:'POST',headers:{'Authorization':'Bearer '+k}});
    o.textContent=await r.text();
    if(r.ok) setTimeout(()=>location.reload(),1500);
  }catch(e){ o.textContent=String(e); }
}
renderTimes();
</script>
</body></html>`,
		html.EscapeString(enabled),
		html.EscapeString(st.Version), html.EscapeString(st.Model),
		html.EscapeString(st.Timezone), html.EscapeString(strings.Join(st.Times, " / ")),
		html.EscapeString(next), html.EscapeString(running),
		html.EscapeString(st.Timezone),
		html.EscapeString(banner),
		rows.String(),
		lastBlock,
		string(timesJSON),
	)
}

func preferID(a runstate.AccountView) string {
	if a.Email != "" {
		return a.Email
	}
	if a.Name != "" {
		return a.Name
	}
	return a.AuthIndex
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

func formatWindow(w *hostapi.QuotaWindow) string {
	if w == nil {
		return "—"
	}
	parts := []string{}
	if w.Remaining != nil {
		parts = append(parts, fmt.Sprintf("剩 %.4g", *w.Remaining))
	}
	if w.Used != nil {
		parts = append(parts, fmt.Sprintf("用 %.4g", *w.Used))
	}
	main := "—"
	if len(parts) > 0 {
		main = strings.Join(parts, " / ")
	}
	reset := ""
	if w.ResetsAt != nil && !w.ResetsAt.IsZero() {
		reset = `<small>重置 ` + html.EscapeString(w.ResetsAt.Format("01-02 15:04")) + `</small>`
	}
	if main == "—" && reset == "" {
		return "—"
	}
	return main + reset
}
