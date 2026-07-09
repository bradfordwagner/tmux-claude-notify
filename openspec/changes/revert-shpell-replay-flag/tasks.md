## 1. Revert and update keybinding

- [x] 1.1 In `tmux-claude-notify.tmux`, build a `CMD` variable: `CMD="cd $PLUGIN_DIR && bin/claude-notify"`.
- [x] 1.2 Change `$GRIMOIRE_SHPELL standard cn '$BINARY' --replay` to `$GRIMOIRE_SHPELL standard cn '$CMD'` (drop `--replay`, invoke via `cd` + relative path instead of the absolute `$BINARY` path).

## 2. Update spec

- [x] 2.1 Apply the `tpm-entry-point` delta spec to `openspec/specs/tpm-entry-point/spec.md`: update the "Keybinding is configurable via TPM option" requirement text and scenarios to the `cd '<plugin-dir>' && bin/claude-notify` wording (no `--replay`), add the "Manual restart from the cn pane uses a relative path" scenario, and remove the "Idle cn window relaunches dashboard on keypress" scenario.

## 3. Verify

- [x] 3.1 `grep -n "replay" tmux-claude-notify.tmux openspec/specs/tpm-entry-point/spec.md` returns no matches.
- [x] 3.2 Confirm `architecture.md` line ~35 reflects the invocation (update if it should show the `cd`+relative-path form).
- [x] 3.3 Reload the TPM plugin (or re-source `tmux-claude-notify.tmux`) and toggle the shpell keybinding a few times in a row to confirm no duplicate/nested `cn` popup appears.
- [x] 3.4 Kill the dashboard process inside an open `cn` pane (drop to a bare shell) and confirm `bin/claude-notify` (relative, no path) successfully restarts it.
