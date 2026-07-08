## 1. Update Data Model

- [x] 1.1 In `ResurrectPane`, replace `WindowIndex int` with `WindowName string` (json tag `window_name`)
- [x] 1.2 Bump `currentVersion` from `1` to `2`

## 2. Update listAllPanes

- [x] 2.1 Change tmux format string from `#{window_index}` to `#{window_name}` in `listAllPanes()`
- [x] 2.2 Replace `livePane.windowIndex int` field with `windowName string`
- [x] 2.3 Remove `strconv.Atoi` for the window field (now a string, no conversion needed)

## 3. Update Save

- [x] 3.1 In `Save()`, record `WindowName: p.windowName` instead of `WindowIndex: p.windowIndex`

## 4. Update Restore

- [x] 4.1 Replace `posKey` struct (session, windowIndex, paneIndex) with `nameKey` struct (session, windowName, paneIndex)
- [x] 4.2 Add early return in `Restore()` when `state.Version < 2`
- [x] 4.3 Build `liveByName` map keyed by `nameKey` instead of `liveByPos` keyed by `posKey`
- [x] 4.4 Remove the `cd <path> && claude --resume` fallback; add a `continue` when `live.currentPath != saved.ProjectPath`

## 5. Update Specs

- [x] 5.1 Archive the updated `resurrect-save` spec delta into `openspec/specs/resurrect-save/spec.md`
- [x] 5.2 Archive the updated `resurrect-restore` spec delta into `openspec/specs/resurrect-restore/spec.md`

## 6. Build and Verify

- [x] 6.1 Run `go build ./...` and confirm no compile errors
- [x] 6.2 Trigger a manual `claude-notify resurrect save` and confirm sidecar has `version: 2` and `window_name` fields
- [x] 6.3 Confirm the "cn" window does not receive `claude --resume` on next restore cycle
