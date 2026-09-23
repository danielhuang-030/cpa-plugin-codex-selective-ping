# Design: Model row layout (更新模型 button placement)

**Date:** 2026-09-23  
**Repo:** cpa-plugin-codex-selective-ping  
**Mode:** Taste redesign–preserve (management rail only)  
**Status:** Awaiting user review before implementation plan

## Problem

After models load successfully, long model ids make `#model-select` wide. The current `.metric-model` row uses `flex-wrap: wrap`, so「更新模型」(`#refresh-models`) wraps to the next line.

## Design Read

Targeted evolution of the CPA plugin management rail metric for operators. Keep existing warm/dark tokens, serif brand + UI sans, secondary button language. No new design system; no full-page restyle.

**Dials:** `DESIGN_VARIANCE 4` / `MOTION_INTENSITY 2` / `VISUAL_DENSITY 7`

## Decision

**Scheme 1 (approved):**

1. Top row: label「模型」left,「更新模型」button right (nowrap).
2. Second row: `#model-select` full width of the rail metric.

Rejected: same-row nowrap shrink (scheme 2); icon-only refresh (scheme 3).

## Markup

Change the model metric from a single flat flex row to a stacked block while preserving ids/testids/i18n:

```html
<div class="metric metric-model" data-testid="metric-model">
  <div class="metric-model-top">
    <span class="k" data-i18n="rail_model">…</span>
    <button type="button" class="btn-secondary" id="refresh-models"
      data-testid="refresh-models" onclick="refreshModels()"
      data-i18n="refresh_models">…</button>
  </div>
  <select id="model-select" class="v" data-testid="model-select">…</select>
</div>
<p id="model-select-hint" class="hint" hidden></p>
```

## CSS (preserve tokens)

- `.metric.metric-model`: column flex; stretch; gap ~8px; no wrap on the control row.
- `.metric-model-top`: `display:flex; justify-content:space-between; align-items:center; gap:8px; width:100%`.
- `#refresh-models`: `flex:0 0 auto; white-space:nowrap` (keep existing padding/font-size).
- `#model-select`: `width:100%; min-width:0; max-width:none` (drop the old `max-width:58%`).
- Do not change global `.metric` behavior for non-model rows.
- No new colors, radii, or motion.

## Behavior (unchanged)

- `loadModels` / `refreshModels` / `X-Csp-Origin` / hint wiring stay as in v0.2.9.
- Empty Management Key still shows `msgNeedKey` on refresh.

## Tests

- UI string/testid tests still find `#model-select`, `#refresh-models`, `refreshModels(`, MODELS_URL.
- Add assertion that HTML contains `metric-model-top` (or equivalent class) and that `#refresh-models` appears before `#model-select` in document order within the metric (label/button row above select).
- Existing management suite must stay green.

## Out of scope

- Changing i18n copy, button style to primary, icon-only control.
- Redesigning the rest of the rail or main panel.
- Version bump/release until implementation is done and user asks to ship.

## Mock

Visual reference: `docs/superpowers/mocks/model-row-layout-mock.html` (+ PNG). Middle card = scheme 1.

