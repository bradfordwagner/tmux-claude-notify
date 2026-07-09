## Context

`tmux-claude-notify.tmux` binds the dashboard keybinding to `custom_shpell standard cn '<binary>' --replay` when grimoire is installed. `--replay` was added in commit `9a91785` to make grimoire relaunch the dashboard (`clear; bash -c '<binary>'`) when the `cn` pane was idle at a shell prompt instead of showing a blank shell. Since that change, the shpell toggle sometimes opens a duplicate/nested `cn` popup instead of relaunching cleanly in place — a regression in the previously reliable toggle behavior. `custom_shpell` itself lives outside this repo (in the `tmux-grimoire` plugin), so the only way to eliminate the regression from this side is to stop passing the flag that triggers it.

## Goals / Non-Goals

**Goals:**
- Restore the pre-`9a91785` shpell keybinding invocation (no `--replay`).
- Change the invocation to `cd '<plugin-dir>' && bin/claude-notify` so a manual restart from the pane only requires typing `bin/claude-notify`.
- Revert `openspec/specs/tpm-entry-point/spec.md` to match, removing the idle-relaunch requirement/scenario and documenting the new invocation form.
- Leave everything else from `9a91785` (resurrect pane-matching-by-window-name, jump refresh fix) untouched — this change targets only the `--replay` flag and the invocation form.

**Non-Goals:**
- Re-implementing idle-pane relaunch a different way. If a blank `cn` shell after idle timeout becomes a problem again, that's a separate future change.
- Changing `custom_shpell` itself (out of repo scope).

## Decisions

- **Revert rather than patch**: `--replay`'s failure mode (duplicate popup) happens inside grimoire's external script, which this repo can't fix directly. Removing the flag is the smallest, safest change and matches the user's request to "revert the change and clean it up." Alternative considered: keep `--replay` but add a guard/debounce in `tmux-claude-notify.tmux` — rejected because the bug lives in `custom_shpell`'s handling of the flag, not in anything this repo controls, so a same-repo workaround can't reliably fix it.
- **No Go code changes**: confirmed via `git log -S "replay" -- cmd/claude-notify/main.go` that the binary never parsed `--replay` — it was purely an argument forwarded to `custom_shpell`. Nothing to clean up on the binary side.
- **No architecture.md change needed**: `architecture.md` line 35 already shows the invocation without `--replay` (it was never updated when the flag was added), so it already reflects the reverted state and needs no edit.
- **`cd` + relative path over absolute path**: passing `cd '$PLUGIN_DIR' && bin/claude-notify` (instead of `'$BINARY'`, the absolute path) means the pane's shell already sits in the plugin directory. If the dashboard process exits and drops back to a bare shell, the user can restart it by typing the short `bin/claude-notify` rather than recalling or retyping the full absolute path. `$PLUGIN_DIR` is computed once via `$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)` and is not expected to contain spaces or shell metacharacters (same trust assumption already made for `$BINARY`), so single-quoting the `cd` argument alone is sufficient — the compound command is not further quoted as a whole, matching how `$BINARY` was passed today.

## Risks / Trade-offs

- [Losing idle-relaunch convenience] → Users who leave the `cn` pane idle at a shell prompt will again see a blank shell instead of an auto-relaunched dashboard until they manually invoke `<binary>`. Mitigation: this is the previously-accepted behavior before `9a91785`; the duplicate-popup regression is worse than the inconvenience being reverted.
