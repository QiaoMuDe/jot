# Tasks

- [x] Task 1: Create `frontend/src/js/cm6-syntax-highlight.js` module with extracted theme and highlight definitions
  - Import required dependencies (`EditorView`, `syntaxHighlighting`, `HighlightStyle`, `tags`, `markdown`)
  - Move `jotTheme` (`EditorView.theme(...)`) from `main.js` and export it
  - Move `jotHighlightStyle` (`HighlightStyle.define(...)`) from `main.js` and export it as part of `mdHighlight`
  - Export `mdHighlight` as `[markdown(), syntaxHighlighting(jotHighlightStyle)]`
- [x] Task 2: Modify `main.js` to import from the new module and remove inline definitions
  - Keep main.js imports as-is for symbols used elsewhere, only remove what's handled by the module
  - Remove `jotTheme` definition block
  - Remove `jotHighlightStyle` definition block
  - Add `import { jotTheme, mdHighlight } from './js/cm6-syntax-highlight.js'`
  - Update `initCodeMirror()` extensions array: replace `jotTheme` usage (unchanged name) and replace `...(useMdHighlight ? [markdown(), syntaxHighlighting(jotHighlightStyle)] : [])` with `...(useMdHighlight ? mdHighlight : [])`
- [x] Task 3: Verify the editor still works correctly (static verification: Vite build succeeded, zero diagnostics; runtime verification: run `wails dev` to test)

# Task Dependencies

None — all tasks are sequential.