## MODIFIED Requirements

### Requirement: Keybinding is configurable via TPM option
The keybinding that opens the dashboard SHALL be read from the TPM option `@claude-notify-key`, defaulting to `C-M-p` if not set. When grimoire is installed, the keybinding SHALL invoke `custom_shpell standard cn '<binary>'`, using `cn` as the shpell window name.

#### Scenario: Default keybinding used when option not set
- **WHEN** `@claude-notify-key` is not set in `tmux.conf`
- **THEN** `C-M-p` is bound to run `custom_shpell standard cn '<binary>'`

#### Scenario: Custom keybinding used when option is set
- **WHEN** `set -g @claude-notify-key 'C-M-n'` is present in `tmux.conf`
- **THEN** `C-M-n` is bound to run `custom_shpell standard cn '<binary>'` instead

#### Scenario: Shpell window name is cn
- **WHEN** the keybinding is triggered and grimoire creates the popup
- **THEN** the placeholder window in the user's session is named `cn`
- **AND** the window in `_shpell-session` is named `cn`
