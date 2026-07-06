## Context

When `claude-notify notify` fires via the Stop hook, the notified pane is often already the active, focused pane — the user is looking right at claude's output. In that case the persistent visual pop is noise: the window tab highlight and background color stay on indefinitely, requiring a manual dashboard trip to dismiss. The fix is: if the pane is already focused at notify time, schedule an auto-clear after a configurable delay (default 15s).

The delay is exposed as TPM option `@claude-notify-active-reset-seconds` (default `15`; `0` disables auto-reset). The binary reads the option at notify time via `tmux show-option -gqv`.

## Goals / Non-Goals

**Goals:**
- Auto-clear window highlight, pane pop, and JSONL entry after N seconds when the notified pane is currently focused.
- Make the delay configurable via `@claude-notify-active-reset-seconds`; `0` = disabled.
- Zero impact on the existing persistent-notification path (non-focused pane).

**Non-Goals:**
- Detecting focus changes during the countdown (if user switches away and back, no re-trigger).
- Any change to the dashboard's manual-clear flow.
- Auto-reset for non-active panes.

## Decisions

### Decision: Detached subprocess for the timer

The Stop hook runs `claude-notify notify` as a synchronous command and waits for it to exit. A goroutine sleeping 15s inside the hook process would block the hook. Instead, `notify` forks a detached subprocess — `claude-notify auto-reset --pane <id> --delay <N>` — using `os.StartProcess` with `Setsid: true` (new session, no controlling terminal), then exits immediately. The subprocess owns the sleep + clear.

*Alternative considered*: tmux `run-shell` with `sleep 15 && ...` via a shell command. Rejected: harder to cancel, no idempotency check.

### Decision: Idempotency check in the subprocess

Before clearing, the subprocess re-reads the JSONL store. If the entry was already cleared (user dismissed via dashboard during the countdown) the subprocess exits without touching tmux styles. This prevents a race where manual dismissal + auto-reset both run.

### Decision: Active-pane detection via tmux display-message

At notify time, after the JSONL write, check `tmux display-message -t "$TMUX_PANE" -p "#{pane_active}#{window_active}"`. If both are `1`, the pane is currently visible and focused. Only then fork the auto-reset subprocess.

*Alternative considered*: check only `pane_active`. Rejected: a pane can be active within a background window — the user isn't looking at it.

### Decision: TPM option read in Go, not in .tmux entry point

The 15s default and the option name (`@claude-notify-active-reset-seconds`) live entirely in the Go binary. The `.tmux` entry point stays thin (per `tpm-entry-point` spec). No new logic is added to the shell file.

## Risks / Trade-offs

- **Subprocess leak if tmux dies**: If tmux exits while the auto-reset subprocess is sleeping, it wakes up and finds no tmux session. `run("set-option", ...)` will fail — non-fatal because all tmux option ops are already best-effort. JSONL file write will succeed but is harmless.
  → Mitigation: subprocess checks `$TMUX` socket exists before any tmux calls.

- **Clock skew / suspend**: If the machine suspends during the 15s sleep, the subprocess wakes late. No consequence beyond a delayed clear.
  → Acceptable.

- **Race: two rapid notify calls, second fires after subprocess starts**: The subprocess does an idempotency check before clearing, so the second notification's entry will be present and the first subprocess will clear it correctly. If the user dismisses manually between the two, the check sees no entry and skips.
  → Covered by the idempotency check.

## Migration Plan

No migration needed. New TPM option defaults to `15`; existing installs auto-inherit the behavior. Users who want the old always-persistent behavior set `set -g @claude-notify-active-reset-seconds 0` in `tmux.conf`.
