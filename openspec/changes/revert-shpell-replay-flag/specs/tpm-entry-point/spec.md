## MODIFIED Requirements

### Requirement: Keybinding is configurable via TPM option
The keybinding that opens the dashboard SHALL be read from the TPM option `@claude-notify-key`, defaulting to `C-M-p` if not set. When grimoire is installed, the keybinding SHALL invoke `custom_shpell standard cn` with a command that changes into the plugin directory and runs the binary by relative path (`cd '<plugin-dir>' && bin/claude-notify`), using `cn` as the shpell window name.

#### Scenario: Default keybinding used when option not set
- **WHEN** `@claude-notify-key` is not set in `tmux.conf`
- **THEN** `C-M-p` is bound to run `custom_shpell standard cn` with command `cd '<plugin-dir>' && bin/claude-notify`

#### Scenario: Custom keybinding used when option is set
- **WHEN** `set -g @claude-notify-key 'C-M-n'` is present in `tmux.conf`
- **THEN** `C-M-n` is bound to run `custom_shpell standard cn` with command `cd '<plugin-dir>' && bin/claude-notify` instead

#### Scenario: Shpell window name is cn
- **WHEN** the keybinding is triggered and grimoire creates the popup
- **THEN** the placeholder window in the user's session is named `cn`
- **AND** the window in `_shpell-session` is named `cn`

#### Scenario: Manual restart from the cn pane uses a relative path
- **WHEN** the dashboard process exits and leaves a bare shell in the `cn` pane
- **THEN** the shell's working directory is already the plugin directory
- **AND** the user can restart the dashboard by running `bin/claude-notify` (no absolute path needed)

Note: this MODIFIED requirement drops the `--replay` flag and the "Idle cn window relaunches dashboard on keypress" scenario introduced in commit `9a91785`. **Reason**: `--replay` caused `custom_shpell` to sometimes open a duplicate/nested `cn` popup instead of cleanly relaunching the dashboard, regressing the previously reliable shpell toggle. **Migration**: None. If the `cn` pane is left idle at a shell prompt, it no longer auto-relaunches the dashboard — manually run `bin/claude-notify` in the pane (now a short relative command, see the scenario above), or close and reopen the shpell via the keybinding.
