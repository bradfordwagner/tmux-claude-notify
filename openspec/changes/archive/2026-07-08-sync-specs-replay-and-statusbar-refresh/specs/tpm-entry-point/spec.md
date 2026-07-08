## MODIFIED Requirements

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
