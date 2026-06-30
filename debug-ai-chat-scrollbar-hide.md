# Debug Session: AI Chat Code Block Scrollbar Hides

## Session Info
- **ID**: `ai-chat-scrollbar-hide`
- **Status**: [OPEN] — waiting for user test
- **Started**: 2026-06-30
- **Problem**: `pre` scrollbar in AI assistant messages appears initially, then disappears after ~1s or on drag

## Hypotheses

1. **~~H1: `overflow` property changes~~** — Tested with `overflow-x: scroll`. The pre now has `overflow-x: scroll` which should ALWAYS show scrollbar.
2. **~~H2: `renderMarkdown` called twice~~** — Checked code, only called once per message.
3. **~~H3: Parent `scrollbar-color` leaks to child~~** — Added explicit `scrollbar-color: var(--scrollbar-thumb) transparent` on pre to override.
4. **H4: Chromium WebView2 overlay scrollbar auto-hide bug** — `overflow-x: scroll` should force classic mode.

## Changes Made

### `frontend/src/css/components/ai-chat.css`
- `overflow-x: auto` → `overflow-x: scroll` (force always visible)
- Added `overflow-y: hidden` (prevent vertical scrollbar interference)
- Added `scrollbar-width: auto; scrollbar-color: var(--scrollbar-thumb) transparent`
- Explicit `::-webkit-scrollbar-*` styles for pre
- Removed `display: none !important` from `::-webkit-scrollbar-button` (potential bug trigger removed)

### `frontend/src/css/scrollbar.css`
- Global `::-webkit-scrollbar-thumb` has `background: var(--scrollbar-thumb)`

### `frontend/src/js/ai-chat.js`
- Added `pre.addEventListener('scroll', (e) => e.stopPropagation(), { passive: true })`

### `frontend/src/js/main.js`
- Added `if (e.target !== container) return;` in scroll event handler

## Test
- [ ] User test: check if scrollbar stays visible with `overflow-x: scroll`
- [ ] If YES: change back to `overflow-x: auto` and test if it still works
- [ ] If NO: scrollbar issue is deeper, possibly WebView2 rendering bug

## Findings
- All CSS changes applied. User still reports scrollbar disappearing.

## Fix
- Changed `overflow-x: auto` → `overflow-x: scroll` as ultimate test.

## Status
- Waiting for user to test with `overflow-x: scroll`.
