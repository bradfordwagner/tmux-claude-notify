## Context

The grimoire `custom_shpell` command takes a name argument that becomes the tmux window name for both the placeholder window in the user's session and the live window in `_shpell-session`. Currently that name is `claude-notify`, which is long and appears in tmux window lists and capture-pane output. Shortening it to `cn` reduces visual clutter and makes scripting easier.

## Goals / Non-Goals

**Goals:**
- Change the shpell window name from `claude-notify` to `cn` in the TPM entry-point keybinding
- Update all documentation examples that reference the old window name

**Non-Goals:**
- Renaming the binary (`bin/claude-notify`), the Go module, or the plugin directory
- Changing any TPM option names (`@claude-notify-*`)
- Altering `_shpell-session` itself (owned by grimoire, not this plugin)

## Decisions

**Single-file change for the window name**: The shpell name is only specified in one place — the `run-shell` argument in `tmux-claude-notify.tmux`. No Go binary code encodes this name, so no binary changes are needed.

**`cn` over alternatives like `notify` or `c`**: Short enough to be unobtrusive, still recognizable as related to the plugin, and avoids clashing with common tmux window names.

## Risks / Trade-offs

- [Existing users with scripts targeting `claude-notify` window name will break] → Mitigation: document the rename in CHANGELOG or README; the change is cosmetic and affects only users who script against the window name directly
- [The CLAUDE.md buffer-capture examples hardcode `claude-notify`] → Mitigation: update those examples as part of this change

## Migration Plan

1. Change the one argument in `tmux-claude-notify.tmux`
2. Update `CLAUDE.md` layout diagram and capture-pane examples
3. Update `architecture.md` if it references the window name
4. Re-source tmux config (or reload TPM) for the change to take effect — no session restart needed
