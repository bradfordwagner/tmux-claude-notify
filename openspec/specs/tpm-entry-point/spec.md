# Spec: tpm-entry-point

## Purpose

`tmux-claude-notify.tmux` is the TPM plugin entry point. It stays thin: compile the binary if stale, register the keybinding. All notification and clear logic lives in the binary.

## Requirements

### Requirement: TPM entry point compiles the binary on load
`tmux-claude-notify.tmux` SHALL compile `bin/claude-notify` from source using `go build` if the binary is missing or the source is newer than the binary.

#### Scenario: Binary compiled successfully on first load
- **WHEN** TPM sources `tmux-claude-notify.tmux` and no binary exists
- **THEN** `go build` runs and produces `bin/claude-notify`

#### Scenario: Binary not recompiled when up to date
- **WHEN** the binary exists and source files are not newer than the binary
- **THEN** `go build` is skipped

#### Scenario: Build failure surfaces as tmux message
- **WHEN** `go build` exits non-zero for any reason
- **THEN** `tmux-claude-notify.tmux` displays an error via `tmux display-message -d 0`
- **AND** exits non-zero so TPM registers the failure
- **AND** the keybinding is NOT registered

### Requirement: Keybinding is configurable via TPM option
The keybinding that opens the dashboard SHALL be read from the TPM option `@claude-notify-key`, defaulting to `C-M-p` if not set. When grimoire is installed, the keybinding SHALL invoke `custom_shpell standard cn '<binary>' --replay`, using `cn` as the shpell window name and `--replay` to ensure the dashboard is relaunched when the "cn" pane is idle.

#### Scenario: Default keybinding used when option not set
- **WHEN** `@claude-notify-key` is not set in `tmux.conf`
- **THEN** `C-M-p` is bound to run `custom_shpell standard cn '<binary>' --replay`

#### Scenario: Custom keybinding used when option is set
- **WHEN** `set -g @claude-notify-key 'C-M-n'` is present in `tmux.conf`
- **THEN** `C-M-n` is bound to run `custom_shpell standard cn '<binary>' --replay` instead

#### Scenario: Shpell window name is cn
- **WHEN** the keybinding is triggered and grimoire creates the popup
- **THEN** the placeholder window in the user's session is named `cn`
- **AND** the window in `_shpell-session` is named `cn`

#### Scenario: Idle cn window relaunches dashboard on keypress
- **WHEN** the "cn" window already exists in the session but its pane is running a shell (`bash`, `zsh`, or `fish`)
- **THEN** `--replay` causes shpell to send `clear; bash -c '<binary>'` to the pane, relaunching the dashboard
- **AND** the popup opens showing the dashboard rather than a blank shell

### Requirement: Entry point stays thin
`tmux-claude-notify.tmux` SHALL only compile the binary and register keybindings. All other logic SHALL live in the binary. The entry point SHALL append a `status-right` segment by default — this is the only permitted `status-right` modification, and it can be suppressed by setting `@claude-notify-statusline 0`.

#### Scenario: No hook registration at TPM load time
- **WHEN** `tmux-claude-notify.tmux` runs
- **THEN** it does NOT register any `pane-focus-in` or `window-linked` hooks
- **AND** it does NOT modify `status-left` or any other global tmux options

#### Scenario: status-right appended by default
- **WHEN** `@claude-notify-statusline` is absent from `tmux.conf`
- **THEN** `tmux-claude-notify.tmux` appends the `claude-notify status` segment to `status-right`

#### Scenario: status-right not modified when explicitly disabled
- **WHEN** `@claude-notify-statusline` is set to `0`
- **THEN** `tmux-claude-notify.tmux` does NOT modify `status-right`

### Requirement: Auto-reset delay is configurable via @claude-notify-active-reset-seconds
The TPM option `@claude-notify-active-reset-seconds` SHALL control how long (in seconds) to wait before auto-clearing a notification on the currently focused pane. The option is read by the Go binary at notify time via `tmux show-option -gqv`; no logic is added to the `.tmux` entry point.

#### Scenario: Option not set — binary uses default of 15 seconds
- **WHEN** `@claude-notify-active-reset-seconds` is absent from `tmux.conf`
- **THEN** `tmux show-option -gqv "@claude-notify-active-reset-seconds"` returns empty
- **AND** the binary defaults to 15 seconds

#### Scenario: Option set in tmux.conf — binary reads custom value
- **WHEN** `set -g @claude-notify-active-reset-seconds 30` is in `tmux.conf`
- **THEN** the binary reads `30` and uses that as the auto-reset delay

#### Scenario: Entry point stays thin — no new shell logic
- **WHEN** `tmux-claude-notify.tmux` runs
- **THEN** it does NOT read or validate `@claude-notify-active-reset-seconds`
- **AND** no logic related to auto-reset is added to the shell entry point
