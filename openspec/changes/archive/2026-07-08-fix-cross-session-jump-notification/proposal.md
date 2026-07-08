## Why

Jumping to a notification from the dashboard (Notifications or Sessions view, `enter`/`w`) silently fails to cross tmux sessions: the popup closes but the outer terminal is left on whatever session/window it was already showing, instead of switching to the notified pane's session. This is the exact behavior `SwitchOuterClientToSessionWindow` and its `always-on-dashboard` spec scenarios ("different session") were supposed to guarantee after the prior `cross-session-jump` fix — but the underlying `tmux display-message` calls used to identify "the outer client" resolve ambiguously when invoked from the dashboard's background process, so the switch is silently skipped or targets the wrong client.

## What Changes

- Fix `outerClientName()` and `SwitchOuterClientToSessionWindow()` in `internal/tmux/tmux.go` to pin an explicit `-t` target (the dashboard's own `$TMUX_PANE`, already exposed via `tmuxclient.PaneID()`) on every `display-message` call used to resolve the current client/session, instead of relying on tmux's implicit "current client" resolution
- This matches the pattern used by every other `display-message` call in the file (`WindowID`, `PanePath`, `Session`, etc., all of which already pass an explicit target) and the explicit workaround already documented in the sibling `tmux-grimoire` plugin's `shpell.sh` ("Use an explicit target ... so pane/session formats resolve from scripts or hooks")
- No behavior change for the same-session case, no new config, no new keybindings

## Capabilities

### New Capabilities
<!-- none: this is a bug fix, not a new capability -->

### Modified Capabilities
- `always-on-dashboard`: the "different session" enter/w navigation scenarios are clarified to require resolving the outer client via an explicit pane target rather than tmux's ambiguous implicit current-client lookup, so cross-session jumps from the dashboard are reliable rather than silently no-op.

## Impact

- `internal/tmux/tmux.go` — `outerClientName()` and `SwitchOuterClientToSessionWindow()`: pass `-t <TMUX_PANE>` explicitly on the `display-message` calls
- No JSONL schema, config, or keybinding changes
- Existing same-session dashboard navigation and the standalone `jump` CLI command are unaffected (they don't go through `outerClientName`/`SwitchOuterClientToSessionWindow`)
