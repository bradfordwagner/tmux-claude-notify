## Why

`claude-notify resurrect restore` matches saved panes by `(session, window_index, pane_index)`, but window indices are volatile — they shift whenever windows are added, removed, or reordered between saves. This causes `claude --resume` to be sent to the wrong pane on restore, most visibly the "cn" grimoire placeholder window, which occupies different index slots across sessions.

## What Changes

- **BREAKING** (sidecar format): `ResurrectPane.window_index int` is replaced by `window_name string`; sidecar version bumped from 1 → 2
- `listAllPanes()` captures `#{window_name}` instead of `#{window_index}`
- `Save()` records `window_name` instead of `window_index`
- `Restore()` matches by `(session, window_name, pane_index)` — names are stable across restarts; indices are not
- `Restore()` requires `pane_current_path == project_path` before sending keys — removes the `cd <path> && claude --resume` fallback that allowed wrong-directory panes to be targeted
- `Restore()` skips v1 sidecars (no `window_name` field) to prevent mismatched matches; next save will produce a v2 sidecar

## Capabilities

### New Capabilities

_(none — this is a bug fix to existing capabilities)_

### Modified Capabilities

- `resurrect-save`: positional key changes from `window_index` to `window_name`; sidecar version bumps to 2
- `resurrect-restore`: match key changes from `(session, window_index, pane_index)` to `(session, window_name, pane_index)`; path mismatch now skips the pane rather than prepending `cd`; v1 sidecars are skipped entirely

## Impact

- `internal/resurrect/resurrect.go`: all changes contained here
- `~/.local/share/tmux-claude-notify/resurrect.json`: stale v1 sidecars are inert until the next save rewrites them as v2; no manual migration needed
