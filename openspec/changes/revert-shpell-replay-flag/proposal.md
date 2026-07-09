## Why

The `--replay` flag was added to the shpell keybinding (commit `9a91785`, "fix idle shpell relaunch") to relaunch the dashboard when the `cn` pane was found idle at a shell prompt. In practice it sometimes causes grimoire's `custom_shpell` to open a duplicate/nested popup on top of the existing `cn` popup instead of cleanly relaunching the dashboard in place. The shpell toggle behavior was reliable before this flag was introduced, so the safest fix is to remove `--replay` and restore the prior invocation.

## What Changes

- Remove the `--replay` argument from the `custom_shpell standard cn '<binary>'` invocation in `tmux-claude-notify.tmux`.
- Change the shpell command from invoking the binary by absolute path (`'<binary>'`) to `cd '<plugin-dir>' && bin/claude-notify` — the pane's shell starts inside the plugin directory, so if the dashboard ever exits back to a bare shell, the user can restart it with the short relative command `bin/claude-notify` instead of typing the full absolute path.
- Revert `openspec/specs/tpm-entry-point/spec.md` requirement text and scenarios back to the pre-`--replay` wording (minus `--replay`), and update it to document the new `cd`-then-relative-path invocation, removing the "Idle cn window relaunches dashboard on keypress" scenario.
- No Go code changes: `--replay` was never parsed by `cmd/claude-notify/main.go` (confirmed via `git log -S "replay" -- cmd/claude-notify/main.go`), so this is purely a keybinding/spec revert with no binary-side cleanup required.
- `architecture.md` already omits `--replay` from the shpell invocation shown on line 35 (it was never updated when the flag was added), so no architecture diagram change is needed there beyond reflecting the `cd`+relative-path form if desired.

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `tpm-entry-point`: the "Keybinding is configurable via TPM option" requirement and its scenarios revert to invoking `custom_shpell standard cn` without `--replay`, and are updated to invoke `cd '<plugin-dir>' && bin/claude-notify` instead of the absolute binary path; the idle-relaunch scenario is removed.

## Impact

- `tmux-claude-notify.tmux`: keybinding invocation changes from `'$BINARY' --replay` to `cd '$PLUGIN_DIR' && bin/claude-notify`.
- `openspec/specs/tpm-entry-point/spec.md`: revert requirement text/scenarios added in `9a91785`, updated for the new `cd`+relative-path invocation.
- No runtime/binary behavior changes — `--replay` was inert from the Go binary's perspective, and `cd && bin/claude-notify` still resolves to the same compiled binary.
- Users who reload the TPM plugin will get the simpler, previously-reliable shpell invocation. A manual restart after the dashboard exits is now just `bin/claude-notify` from the pane's starting directory. The idle-pane-relaunch convenience is intentionally given up in favor of restoring popup reliability.
