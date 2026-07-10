# Tasks

- [x] Task 1: Fix delete animation CSS cascade conflict
  - File: `frontend/src/main.js` ([line 2623](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L2623))
  - In `deleteProfile()` function, before adding `preset-delete-out`, remove the conflicting `preset-row-enter` class
  - Change: `rowEl.classList.add('preset-delete-out')` → `rowEl.classList.remove('preset-row-enter'); rowEl.classList.add('preset-delete-out');`
  - This ensures `presetDeleteOut` animation plays and `animationend` event fires

- [x] Task 2: Fix scrollbar disappearing by adding standalone max-height
  - File: `frontend/src/css/components/settings-panel.css` ([line 733](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/settings-panel.css#L733))
  - Add `max-height: 500px` as a regular CSS declaration to `.preset-mgr-list`
  - This ensures height limit persists regardless of animation fill state

## Task Dependencies

- [Task 1] depends on none
- [Task 2] depends on none
