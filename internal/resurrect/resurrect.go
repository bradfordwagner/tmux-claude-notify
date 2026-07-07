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

const currentVersion = 1

type ResurrectPane struct {
	TmuxSession string `json:"tmux_session"`
	WindowIndex int    `json:"window_index"`
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
	windowIndex int
	paneIndex   int
	paneID      string
	currentCmd  string
	currentPath string
}

func listAllPanes() []livePane {
	out, err := exec.Command("tmux", "list-panes", "-a", "-F",
		"#{session_name}\t#{window_index}\t#{pane_index}\t#{pane_id}\t#{pane_current_command}\t#{pane_current_path}").Output()
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
		wi, _ := strconv.Atoi(parts[1])
		pi, _ := strconv.Atoi(parts[2])
		panes = append(panes, livePane{
			session:     parts[0],
			windowIndex: wi,
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
			WindowIndex: p.windowIndex,
			PaneIndex:   p.paneIndex,
			PaneID:      p.paneID,
			SessionID:   sessionID,
			ProjectPath: p.currentPath,
		})
	}
	return state.Write()
}

type posKey struct {
	session     string
	windowIndex int
	paneIndex   int
}

// Restore reads the resurrect sidecar and replays "claude --resume <id>" into
// each matching live pane. Panes already running claude* are skipped.
func Restore() error {
	state, err := Load()
	if err != nil || len(state.Panes) == 0 {
		return err
	}

	panes := listAllPanes()
	liveByPos := make(map[posKey]livePane, len(panes))
	for _, p := range panes {
		liveByPos[posKey{p.session, p.windowIndex, p.paneIndex}] = p
	}

	for _, saved := range state.Panes {
		live, ok := liveByPos[posKey{saved.TmuxSession, saved.WindowIndex, saved.PaneIndex}]
		if !ok {
			continue
		}
		if strings.HasPrefix(live.currentCmd, "claude") {
			continue
		}
		cmd := "claude --resume " + saved.SessionID
		if saved.ProjectPath != "" && live.currentPath != saved.ProjectPath {
			cmd = "cd " + saved.ProjectPath + " && " + cmd
		}
		_ = exec.Command("tmux", "send-keys", "-t", live.paneID, cmd, "Enter").Run()
	}
	return nil
}
