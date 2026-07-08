## 1. TPM Entry Point

- [x] 1.1 In `tmux-claude-notify.tmux`, change the shpell name argument from `claude-notify` to `cn` in the `bind-key` call

## 2. Documentation

- [x] 2.1 Update `CLAUDE.md` layout diagram: rename `window: claude-notify` to `window: cn` in both the user session and `_shpell-session` trees
- [x] 2.2 Update `CLAUDE.md` capture-pane examples: replace `_shpell-session:claude-notify` with `_shpell-session:cn` and update the `grep ':claude-notify$'` filter to `grep ':cn$'`
- [x] 2.3 Update `architecture.md` if it contains any node or label referencing the `claude-notify` window name
