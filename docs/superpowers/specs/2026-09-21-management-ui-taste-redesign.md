# Management UI Taste Redesign (issue #8)

Date: 2026-09-21  
Status: **design locked (v3)** — awaiting spec review before plan/TDD  
Repo: `cpa-plugin-codex-selective-ping`  
Mock: `docs/superpowers/mocks/management-ui-mock-v3.html`  
Screenshots: `management-ui-mock-v3-{light,dark,empty}.png`

## 1. Goal

Fully **redesign** the CPA Management page for Codex Selective Ping so it no longer reads as a generic admin card stack. Apply Taste Skill anti-slop discipline to an **ops console** (not a marketing landing): distinctive layout, warm editorial visual language, clearer empty states and CTA hierarchy. **Do not change** ping/whitelist/schedule semantics or API contracts.

## 2. Design Read

> Reading this as: redesign of a CPA plugin management console for a technical operator, with a warm editorial ops language (paper canvas, serif titles, terracotta accent), leaning toward native CSS tokens + left ops rail + rhythm grid + account cards + receipt-style last run — dials **VARIANCE 8 / MOTION 4 / DENSITY 5**.

Rejected: AI-purple mesh, glassmorphism, centered hero, Inter+slate three-card polish that looks like the old page with new colors.

## 3. Decisions (locked)

| Topic | Choice |
|--------|--------|
| Scope | Visual + IA redesign of Management UI only |
| Aesthetic | Full redesign (v3), not product-polish recolor (v2) |
| Implementation path | **A**: in-place rewrite of embedded HTML/CSS in `internal/management/ui.go` (defer extract to separate files) |
| Block order (logical) | Rail (status/actions) + workspace: rhythm → principles → accounts → last run |
| Empty states | Yes — no whitelist / no last run with next-step CTAs |
| Theme | Follow CPA `data-theme` light/dark; warm tokens for both |
| i18n | Keep zh-Hant / en / ja; all new strings keyed |
| Sidebar Menu string | Out of scope (CPA single-string ABI) |
| Behavior | Preserve: schedule enable, times, timezone, account whitelist save, run-now uses **saved** whitelist, last-run persistence, quota columns informational only |

## 4. Information architecture

### 4.1 Left rail (sticky on wide screens)

- Brand: “Selective **Ping**” (serif + italic accent word)
- Live status chip (schedule on/off + version)
- “此刻” metrics: next run, whitelist count, model, timezone
- Management Key field (session only, not persisted in localStorage)
- Primary stack: **儲存設定** (primary) → **立刻執行** (secondary) → **重新整理** (ghost)

### 4.2 Workspace

1. **今天的節奏** — time slots as a grid; mark next upcoming; add timezone / new time / enable checkbox  
2. **操作原則** — short rules (check → save → run-now uses saved list; empty whitelist = ping nobody; quota informational)  
3. **要打誰** — account **cards** (not dense data table): checkbox, identity, plan chip, 5h/weekly bars, status chip; filters All / Selected / Abnormal  
4. **上一輪回報** — receipt summary (dark ink panel) + per-account result rows  

### 4.3 Empty states

- No selection: dashed empty panel + CTA to focus account list  
- No last run: empty panel + CTA run-now  

## 5. Visual system

- Canvas: warm paper (`#f6f1ea` light / deep plum-ink dark)  
- Accent: terracotta (`#c45c26` / lighter on dark) for primary CTA and “next” slot  
- Secondary accent: muted teal for selected account / success-adjacent chrome  
- Type: UI sans for controls; serif for brand + section titles  
- Radius ~16–22px panels; hairline borders; soft warm shadow — no heavy elevation  
- Motion: ≤200ms hover/focus; honor `prefers-reduced-motion`  
- Numbers: tabular nums for times and quotas  

## 6. Technical approach

- Continue server-rendered HTML string from `RenderStatusPage` (or equivalent) in `internal/management/ui.go`  
- Replace layout/CSS to match v3 mock; keep existing JS hooks for save/run/refresh/i18n/theme sync  
- Tests: extend `ui_test.go` (and related) for new landmarks — rail, rhythm slots, empty-state copy keys, primary CTA order — without requiring a browser  
- Run tests via **Docker** (`compose.yaml` / project Makefile) per sp-flow  
- No Cloud Agent  

## 7. Non-goals

- Changing selection / schedule / ping behavior  
- Full run history beyond last run  
- Multilingual CPA sidebar Menu  
- Splitting UI into `embed` files in this issue (follow-up OK)  
- Pixel-perfect CPA host chrome (we only own the plugin page iframe/content)

## 8. Acceptance

1. Side-by-side with v3 mock: same IA (rail + rhythm + cards + receipt); clearly different from pre-#8 UI  
2. Regression: select → save → run-now / schedule; last-run survives reload; i18n + theme OK  
3. Empty states visible when whitelist empty / no last run  
4. `go test` (Docker) green; version bump + release only after implementation (separate step)

## 9. Open follow-ups (not blocking)

- Extract HTML/CSS embed files if `ui.go` becomes unmaintainable  
- Optional denser “table mode” toggle if account count grows large  

## 10. Spec self-review

- [x] No TBD placeholders for locked decisions  
- [x] No contradiction with “redesign vs polish” (v3 redesign locked)  
- [x] Scope limited to UI; behavior preserved  
- [x] Mock path recorded for implementer  
