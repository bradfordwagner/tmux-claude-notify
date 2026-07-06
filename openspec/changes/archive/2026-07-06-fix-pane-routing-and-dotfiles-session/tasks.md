## 1. Store: Add UnclearedForWindow

- [x] 1.1 Add `UnclearedForWindow(windowID string) ([]Record, error)` to `internal/store/store.go` — reads all records and returns uncleared ones whose `Window` field matches the given window ID

## 2. Fix Multi-Pane Window Clear Routing

- [x] 2.1 Update `runClear` in `cmd/claude-notify/main.go`: before clearing window styles, read the window ID from the JSONL record (via `store.ReadAll` + find matching pane), call `store.ClearPane`, then call `store.UnclearedForWindow`; only call `ClearWindowStyle`/`ClearPopStyle` if the result is empty
- [x] 2.2 Remove the unconditional `ClearWindowStyle`/`ClearPopStyle` calls from the previous `runClear` flow

## 3. Fix Dashboard Live-Pane Filter

- [x] 3.1 In `internal/ui/model.go` `loadEntries`: keep building `liveSet` for the "gone" indicator but remove the `liveSet[r.Pane]` guard from the entry inclusion check — all uncleared entries are shown
- [x] 3.2 Update path display in `loadEntries`: when `PanePath` returns an empty string AND the pane is not in `liveSet`, set the entry's `Path` field to `(gone)` instead of an empty string

## 4. Architecture Diagram Update

- [x] 4.1 Update `architecture.md` to document the multi-pane window-level clearing behavior and the gone-pane display path in the dashboard flow

## 5. Fix Pane Pop Targeting

- [x] 5.1 Change `SetPopStyle`/`ClearPopStyle` in `internal/tmux/tmux.go` to use `select-pane -t <paneID> -P` instead of `set-option window-active-style` — pane-scoped, not window-focused-pane-scoped
- [x] 5.2 Update all call sites (`runNotify`, `runClear`, UI enter handler, `applyStateChange`) to pass `paneID` instead of `windowID`; `ClearPopStyle` is always called per-pane, no longer gated on last-in-window
- [x] 5.3 Update delta spec, main spec, and architecture.md to reflect `select-pane -P` behavior

## 6. Verification

- [x] 6.1 Build the binary (`task build` or `go build ./...`) and confirm no compilation errors
- [x] 6.2 Open two panes in the same window, trigger notifications in both, dismiss one from the dashboard, and confirm the window highlight remains
- [x] 6.3 Trigger a notification, close the pane, open the dashboard, and confirm the entry shows `(gone)` in the PATH column and can be cleared
