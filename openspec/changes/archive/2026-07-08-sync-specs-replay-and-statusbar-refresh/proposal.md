## Why

Two behaviour changes were made this session that are not yet reflected in the canonical specs: the dashboard keybinding now passes `--replay` so the TUI relaunches when the "cn" window is idle, and `claude-notify jump` now immediately refreshes the status bar after clearing a notification. The specs for `tpm-entry-point` and `quickjump` must be updated to match the live code.

## What Changes

- `tpm-entry-point` keybinding scenario updated: `custom_shpell` call now includes `--replay` as the 4th argument
- `tpm-entry-point` gains a new scenario describing the idle-relaunch behaviour `--replay` enables
- `quickjump` gains a new scenario: status bar is immediately refreshed via `tmux refresh-client -S` after a jump clears its notification

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- `tpm-entry-point`: keybinding requirement updated to include `--replay`; idle-relaunch scenario added
- `quickjump`: jump-clears-notification requirement updated to include immediate status bar refresh

## Impact

- `openspec/specs/tpm-entry-point/spec.md`
- `openspec/specs/quickjump/spec.md`
- No code changes — implementation is already complete
