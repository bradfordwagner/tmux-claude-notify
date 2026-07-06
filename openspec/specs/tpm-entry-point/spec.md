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
The keybinding that opens the dashboard SHALL be read from the TPM option `@claude-notify-key`, defaulting to `C-M-p` if not set.

#### Scenario: Default keybinding used when option not set
- **WHEN** `@claude-notify-key` is not set in `tmux.conf`
- **THEN** `C-M-p` is bound to `run-shell "bin/claude-notify"`

#### Scenario: Custom keybinding used when option is set
- **WHEN** `set -g @claude-notify-key 'C-M-n'` is present in `tmux.conf`
- **THEN** `C-M-n` is bound to `run-shell "bin/claude-notify"` instead

### Requirement: Entry point stays thin
`tmux-claude-notify.tmux` SHALL only compile the binary and register the keybinding. All other logic SHALL live in the binary.

#### Scenario: No hook registration at TPM load time
- **WHEN** `tmux-claude-notify.tmux` runs
- **THEN** it does NOT register any `pane-focus-in` or `window-linked` hooks
- **AND** it does NOT modify `status-left`, `status-right`, or any global tmux options
