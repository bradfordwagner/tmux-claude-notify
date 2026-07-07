## Context

The existing `auto-reset-active-pane` feature spawns a background subprocess when a notification fires. That subprocess either:
- (A) immediately sleeps `delaySecs` seconds (when the pane was already focused at notify time), or
- (B) polls every 2 seconds until the window becomes focused, *then* sleeps `delaySecs` seconds.

Both paths use the same `delaySecs` value (`@claude-notify-active-reset-seconds`, default 15). Path B produces a 15+ second delay between "user navigates to the popped window" and "notification clears" — too slow to feel responsive.

Current signature: `runAutoReset(paneID string, delaySecs int)`, `forkAutoReset(paneID string, delaySecs int)`, subprocess flag `--delay`.

## Goals / Non-Goals

**Goals:**
- Clear the notification quickly (default 2s) when the user navigates *to* a popped window.
- Keep the existing 15s grace period when the user was *already in* the window when the notification fired.
- Allow either path to be disabled independently (set its delay to `0`).
- No new mechanism required — extend the existing polling subprocess.

**Non-Goals:**
- Instant (0s) clearing — a minimum 1s floor prevents races and avoids clearing during rapid window cycling.
- Detecting pane-level focus (the existing approach uses `window_active` via `IsPaneFocused`; that level of granularity is sufficient and avoids the broken `pane-focus-in` hook).
- Dashboard changes — the dashboard already reads from the store and updates when an entry is cleared.

## Decisions

### Decision: Two independent delay parameters rather than one

**Options considered:**
1. Single configurable delay for all paths — simplest but can't satisfy "fast on navigate, slow on already-focused" without breaking existing setups.
2. Two parameters: `activeResetSecs` (existing `@claude-notify-active-reset-seconds`) for the already-focused path, `navClearSecs` (new `@claude-notify-nav-clear-seconds`) for the poll-detected-focus path.

**Choice: Option 2.**
Users who rely on the 15s grace period (to notice the pop before it disappears) keep that behavior unchanged. Navigation-to-clear gets its own short default (2s) that can be tuned or disabled independently.

### Decision: Fork subprocess if either delay > 0

Currently: subprocess is not spawned when `delaySecs = 0`.
New rule: spawn the subprocess if `activeResetSecs > 0 || navClearSecs > 0`. This allows `active-reset-seconds = 0` (disable "already-focused" clearing) while keeping `nav-clear-seconds = 2` (enable navigation clearing), and vice versa.

### Decision: `navClearSecs` default = 2 seconds

One second would feel snappier but increases the chance of an accidental clear when rapidly cycling through windows. Two seconds gives a comfortable window of "you're actually looking at this pane now". Configurable to 0 (disable) or any positive integer.

### Decision: Pass `navClearSecs` as `--nav-delay` flag on the subprocess

Keeps the subprocess fully self-contained with no TPM option reads at run time — all configuration is resolved in the parent's `notify` subcommand and threaded through as flags. Consistent with how `--delay` works today.

## Risks / Trade-offs

- **Risk: User rapidly cycles through windows and notification clears unintentionally** → Mitigation: 2s default is long enough to survive most cycling; user can raise `@claude-notify-nav-clear-seconds` if needed.
- **Risk: Subprocess proliferation** — two subprocesses forked per notification if both options > 0 is **not** the case here; one subprocess handles both paths sequentially.
- **Trade-off: Both options independently disableable** — adds one extra TPM option and one extra flag, but avoids surprising users who disabled `active-reset-seconds` and now get unexpected nav-clear.

## Migration Plan

- Existing installs: `@claude-notify-active-reset-seconds` behavior unchanged. `@claude-notify-nav-clear-seconds` defaults to 2, which is a visible behavior improvement — pop clears faster when you navigate to the window. No action needed.
- To disable nav-clear only: `set -g @claude-notify-nav-clear-seconds 0` in `tmux.conf`.
- To disable all auto-reset: set both to 0.
- Binary must be rebuilt after update (`task build` or TPM will rebuild on next tmux start).

## Open Questions

None — the scope is well-bounded by the existing auto-reset infrastructure.
