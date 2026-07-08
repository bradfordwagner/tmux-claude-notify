## MODIFIED Requirements

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
