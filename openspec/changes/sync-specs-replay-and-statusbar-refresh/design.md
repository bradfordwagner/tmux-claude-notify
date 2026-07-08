## Context

Both changes are already implemented. This change exists solely to bring the specs into sync with the code.

**`--replay` flag**: `shpell.sh` has a `replay_flag` parameter (4th arg) that gates idle-relaunch logic. Without it, pressing `C-M-p` when "cn" already exists but is idle shows a blank shell instead of the dashboard. Adding `--replay` to the keybinding invocation enables automatic relaunch when the pane is idle (`bash`/`zsh`/`fish`).

**`tmux refresh-client -S`**: After `runJump()` clears a notification, the status bar badge persists until the next `status-interval` tick (default 5s, user's setting also 5s). `refresh-client -S` forces an immediate status-only redraw with no visual flicker.

## Goals / Non-Goals

**Goals:**
- Specs match the live code

**Non-Goals:**
- Any code changes
- Changing the behaviour being documented

## Decisions

No decisions required — this is a spec sync, not a design exercise.

## Risks / Trade-offs

None. Spec-only change.
