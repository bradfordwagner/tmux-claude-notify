## Why

The dashboard is the only way to know how many claude sessions are waiting — you have to actively open it. A status-bar count and a direct-jump keybinding give passive awareness and single-keypress response, eliminating the "open dashboard → select → close" round-trip for the common single-notification case.

## What Changes

- New `status` subcommand outputs a plain-text count string (e.g. `⚡ 2`) suitable for embedding in `status-right` via `#(...)` polling
- New `@claude-notify-statusline` TPM option: when set to a non-empty format string, appends a `status-right` segment using the `status` subcommand
- New `@claude-notify-jump-key` TPM option (default `C-M-P`): binds a key that jumps directly to the oldest waiting pane (by `ts` ascending) without opening the dashboard; clears the notification as the dashboard `enter` handler does
- README Configuration table updated with both new options

## Capabilities

### New Capabilities

- `statusbar-segment`: `status` subcommand + TPM wiring that inserts a live notification-count badge into `status-right`
- `quickjump`: keybinding that resolves the oldest uncleared notification and runs `SelectPane` + `SelectWindow` + clear in one shot

### Modified Capabilities

- `tpm-entry-point`: registers the new `@claude-notify-jump-key` binding and optionally wires `@claude-notify-statusline` into `status-right`

## Impact

- `cmd/claude-notify/main.go` — new `status` subcommand branch
- `internal/store/store.go` — new `OldestUncleared()` helper (returns the entry with the smallest `ts` among uncleared records)
- `tmux-claude-notify.tmux` — reads `@claude-notify-jump-key` and `@claude-notify-statusline`; appends segment if option set
- `README.md` — two new rows in the Configuration table
