## 1. tmux Helper

- [x] 1.1 Add `IsPaneFocused(paneID string) bool` to `internal/tmux/tmux.go` — runs `tmux display-message -t <paneID> -p "#{pane_active}#{window_active}"` and returns true only when result is `"11"`

## 2. Auto-Reset Subcommand

- [x] 2.1 Add `case "auto-reset":` to the switch in `cmd/claude-notify/main.go` that parses `--pane <id>` and `--delay <seconds>` flags
- [x] 2.2 Implement `runAutoReset(paneID string, delaySecs int)` in `cmd/claude-notify/main.go`: sleep the delay, check `store.HasUnclearedPane(paneID)`, if still uncleared call `runClear(paneID)`

## 3. Notify — Active-Pane Detection + Fork

- [x] 3.1 Add `ActiveResetSeconds() int` helper to `internal/tmux/tmux.go` — reads `@claude-notify-active-reset-seconds` via `show-option -gqv`, returns `15` if empty, `0` if value is `"0"`
- [x] 3.2 In `runNotify()` (`cmd/claude-notify/main.go`), after `store.Append`, call `ActiveResetSeconds()` and `IsPaneFocused(paneID)`; if both conditions met (delay > 0 and pane focused), fork a detached `claude-notify auto-reset --pane <id> --delay <N>` subprocess using `os.StartProcess` with `Setsid: true` and closed stdio
- [x] 3.3 Verify the forked process does not block `runNotify` and exits immediately

## 4. Architecture Diagram

- [x] 4.1 Update `architecture.md` to document the auto-reset flow: notify → active-pane check → detached subprocess → sleep → idempotency check → clear
