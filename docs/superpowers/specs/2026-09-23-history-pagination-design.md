# History list frontend pagination — design

**Date:** 2026-09-23  
**Issue:** TBD (open when implementing / shipping)  
**Status:** Awaiting user review of written spec

## Problem

The Management UI「執行與歷史」list renders every persisted run (up to `history_limit`, default 60) as expandable rows. With many runs the page becomes long even though older rows stay collapsed.

## Goal

Add **frontend-only pagination** so the history section shows a manageable page of rows, with a UI-editable page size (default 10) remembered in `localStorage` — without changing plugin config or the status API payload.

## Locked decisions

| Item | Choice |
| --- | --- |
| Layer | **Approach A:** server still renders all history rows; JS shows/hides the current page |
| Page size default | **10** |
| Page size choices | **10 / 20 / 50** |
| Page size persistence | Browser **`localStorage`** key `csp-hist-page-size` (not plugin config) |
| Invalid / missing storage | Fall back to **10** |
| Change page size | Reset to **page 1**, then re-slice |
| Plugin config | **No new fields** |
| `history_limit` / `run_history` API | **Unchanged** (still full list, trim on write) |
| Empty history | No pager (keep existing empty state) |
| Total ≤ page size | Show pager chrome (page 1/1 + size select); prev/next **disabled** |
| Expand behavior | Unchanged `toggleHist`; newest row (index 0) still default-open in HTML; if that row is not on the visible page, do not force-open another row |
| Locales | zh-Hant / en / ja for pager copy |

## Behavior

1. On load, read `localStorage['csp-hist-page-size']`; if not in `{10,20,50}`, use 10.
2. Compute `pageCount = max(1, ceil(itemCount / pageSize))` when `itemCount > 0`.
3. Show only `.hist-item` indices in `[start, start+pageSize)` for the current page; hide the rest (CSS class or `display`).
4. Pager below `.hist`: Previous, “page n / N” (i18n), Next, “per page” label + `<select>` with 10/20/50.
5. Previous disabled on page 1; Next disabled on last page. Buttons `type="button"`.
6. Changing select writes localStorage, sets page to 1, re-applies visibility.
7. Changing page does **not** full-page reload and does **not** re-fetch status.

## UI placement

- Pager sits **below** the history list (after `data-testid="run-history"`).
- Marker: `data-testid="hist-pager"` (and stable hooks for prev/next/select as needed for string tests).

## Implementation surface

- Primarily `internal/management/ui.go` (HTML/CSS/JS), `i18n.go`, `ui_test.go`, `i18n_test.go`.
- JS helper e.g. `initHistPager()` invoked from existing page script bootstrap.
- No changes to `internal/runstate`, `internal/config` history fields, or status JSON shape beyond what already exists.

## Testing (Docker `go test`)

- With ≥1 history run in status fixture: HTML contains `data-testid="hist-pager"` and option values 10/20/50.
- i18n keys present in zh-Hant / en / ja.
- JS source contains localStorage key `csp-hist-page-size`, default 10, and logic that resets to page 1 on size change (string / presence assertions consistent with existing `ui_test` style).
- Empty history: pager absent (or not required).

## Out of scope

- Plugin config `history_page_size` (or any new config field)
- Changing default `history_limit`
- Backend / status API `page` / `page_size` parameters
- URL query or hash for page index
- Virtual scroll / infinite scroll
- Persisting current **page number** across reloads (only page **size** is stored)

## Success criteria

- User sees at most `pageSize` history rows at a time (default 10).
- Can switch 10/20/50; preference survives reload via localStorage.
- Full history still available by paging; disk/API retention still governed by `history_limit` only.
