package resurrect

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	claudepkg "github.com/bradfordwagner/tmux-claude-notify/internal/claude"
)

const currentVersion = 2

type ResurrectPane struct {
	TmuxSession string `json:"tmux_session"`
	WindowName  string `json:"window_name"`
	PaneIndex   int    `json:"pane_index"`
	PaneID      string `json:"pane_id"`
	SessionID   string `json:"session_id"`
	ProjectPath string `json:"project_path"`
}

type ResurrectState struct {
	Version int             `json:"version"`
	Panes   []ResurrectPane `json:"panes"`
}

func dataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "tmux-claude-notify")
}

// FilePath returns the path to the resurrect sidecar JSON file.
func FilePath() string {
	return filepath.Join(dataDir(), "resurrect.json")
}

func Load() (ResurrectState, error) {
	var s ResurrectState
	data, err := os.ReadFile(FilePath())
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(data, &s)
	return s, err
}

func (s ResurrectState) Write() error {
	path := FilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

type livePane struct {
	session     string
	windowName  string
	paneIndex   int
	paneID      string
	currentCmd  string
	currentPath string
}

func listAllPanes() []livePane {
	out, err := exec.Command("tmux", "list-panes", "-a", "-F",
		"#{session_name}\t#{window_name}\t#{pane_index}\t#{pane_id}\t#{pane_current_command}\t#{pane_current_path}").Output()
	if err != nil {
		return nil
	}
	var panes []livePane
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 6)
		if len(parts) != 6 {
			continue
		}
		pi, _ := strconv.Atoi(parts[2])
		panes = append(panes, livePane{
			session:     parts[0],
			windowName:  parts[1],
			paneIndex:   pi,
			paneID:      parts[3],
			currentCmd:  parts[4],
			currentPath: parts[5],
		})
	}
	return panes
}


// Save snapshots all tmux panes currently running claude* into the resurrect sidecar.
// Session ID is always derived from the latest transcript file (ground truth for the
// active session); project path comes from pane_current_path (always authoritative).
func Save() error {
	panes := listAllPanes()

	state := ResurrectState{Version: currentVersion}
	for _, p := range panes {
		if !strings.HasPrefix(p.currentCmd, "claude") {
			continue
		}
		sessionID := claudepkg.LatestTranscriptID(p.currentPath)
		if sessionID == "" {
			continue
		}
		state.Panes = append(state.Panes, ResurrectPane{
			TmuxSession: p.session,
			WindowName:  p.windowName,
			PaneIndex:   p.paneIndex,
			PaneID:      p.paneID,
			SessionID:   sessionID,
			ProjectPath: p.currentPath,
		})
	}
	return state.Write()
}

type nameKey struct {
	session    string
	windowName string
	paneIndex  int
}

// Restore reads the resurrect sidecar and replays "claude --resume <id>" into
// each matching live pane. Matching is by (session, window_name, pane_index) —
// window names are stable across restarts while window indices are not. Panes
// already running claude* or whose current path does not match the saved project
// path are skipped. Sidecars from version < 2 (without window_name) are ignored.
func Restore() error {
	state, err := Load()
	if err != nil || len(state.Panes) == 0 {
		return err
	}
	if state.Version < 2 {
		return nil
	}

	panes := listAllPanes()
	liveByName := make(map[nameKey]livePane, len(panes))
	for _, p := range panes {
		liveByName[nameKey{p.session, p.windowName, p.paneIndex}] = p
	}

	for _, saved := range state.Panes {
		live, ok := liveByName[nameKey{saved.TmuxSession, saved.WindowName, saved.PaneIndex}]
		if !ok {
			continue
		}
		if strings.HasPrefix(live.currentCmd, "claude") {
			continue
		}
		if live.currentPath != saved.ProjectPath {
			continue
		}
		cmd := "claude --resume " + saved.SessionID
		_ = exec.Command("tmux", "send-keys", "-t", live.paneID, cmd, "Enter").Run()
	}
	return nil
}
