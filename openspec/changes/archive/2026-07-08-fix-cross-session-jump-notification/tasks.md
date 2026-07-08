## 1. Fix ambiguous client resolution

- [x] 1.1 In `internal/tmux/tmux.go`, update `outerClientName()` to call `display-message -p -t <TMUX_PANE> "#{client_name}"` instead of the untargeted call, using `PaneID()` (or the existing `$TMUX_PANE` env read) to obtain the pane ID
- [x] 1.2 Update `SwitchOuterClientToSessionWindow()` to call `display-message -p -t <TMUX_PANE> "#{client_session}"` instead of the untargeted call
- [x] 1.3 Guard against an empty `$TMUX_PANE` (return early / no-op) consistent with existing error handling in both functions

## 2. Verify

- [x] 2.1 `go build ./...` and `go vet ./...` pass
- [x] 2.2 Manually reproduce in a real tmux session: two sessions ("main", "edit"), a `claude` pane waiting in "edit", open the dashboard via `C-M-p` from "main", press `enter` on the notification, confirm the outer terminal lands on the correct session/window in "edit" after the popup closes — confirmed by user (k8s → edit)
- [ ] 2.3 Manually verify the same-session case and the `w` (Sessions view) cross-session case are unaffected/still work
- [x] 2.4 Confirm the standalone `claude-notify jump` keybinding (unaffected by this change) still works across sessions — confirmed by user

## 3. Close out

- [x] 3.1 Confirm no architecture.md diagram update is needed (no data-flow, component-boundary, or hook-wiring change — pure client-resolution bug fix)
- [ ] 3.2 Run `/opsx:archive` once merged
