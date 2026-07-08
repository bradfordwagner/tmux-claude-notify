## 1. Update tpm-entry-point spec

- [x] 1.1 In `openspec/specs/tpm-entry-point/spec.md`, update keybinding requirement text: `custom_shpell standard cn '<binary>'` → `custom_shpell standard cn '<binary>' --replay`
- [x] 1.2 Update Default keybinding scenario to reference `--replay`
- [x] 1.3 Update Custom keybinding scenario to reference `--replay`
- [x] 1.4 Add "Idle cn window relaunches dashboard on keypress" scenario

## 2. Update quickjump spec

- [x] 2.1 In `openspec/specs/quickjump/spec.md`, update requirement description to mention `tmux refresh-client -S`
- [x] 2.2 Update "Single notification waiting" scenario to include immediate status bar refresh
- [x] 2.3 Add "Status bar refreshes immediately after jump" scenario
- [x] 2.4 Update "Jump clears notification identically to dashboard enter" requirement to include status bar refresh
