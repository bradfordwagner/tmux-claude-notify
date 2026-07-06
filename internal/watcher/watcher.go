package watcher

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const tailN = 20

// Status represents the derived agent state from transcript analysis.
type Status string

const (
	StatusRunning Status = "running"
	StatusWaiting Status = "waiting"
	StatusStale   Status = "stale"
)

// StateChange is emitted whenever a tracked transcript transitions state.
// Clear=true means the user responded and the notification should be removed.
type StateChange struct {
	PaneID   string
	WindowID string
	Session  string
	Status   Status
	Clear    bool
}

type paneState struct {
	paneID     string
	windowID   string
	session    string
	transcript string
	staleTimer *time.Timer
}

// Watcher watches Claude Code transcript files embedded in the dashboard TUI
// process. It discovers panes running claude*, maps them to transcript files,
// and emits StateChange events when transcript state changes.
type Watcher struct {
	changes  chan StateChange
	done     chan struct{}
	fw       *fsnotify.Watcher
	panes    map[string]*paneState // transcript path → paneState
	mu       sync.Mutex
	staleDur time.Duration
}

func New() (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		changes:  make(chan StateChange, 16),
		done:     make(chan struct{}),
		fw:       fw,
		panes:    make(map[string]*paneState),
		staleDur: staleDuration(),
	}, nil
}

// Changes returns the channel on which StateChange events are delivered.
func (w *Watcher) Changes() <-chan StateChange {
	return w.changes
}

// Start discovers active claude panes and begins watching their transcripts.
func (w *Watcher) Start() {
	w.discover()
	go w.run()
}

// Close stops the watcher and releases all resources.
func (w *Watcher) Close() {
	close(w.done)
	w.fw.Close()
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, ps := range w.panes {
		if ps.staleTimer != nil {
			ps.staleTimer.Stop()
		}
	}
}

// Reconcile scans active transcripts and returns the current state for each
// tracked pane. Call this on dashboard open to correct stale JSONL entries.
func (w *Watcher) Reconcile() []StateChange {
	w.discover()
	w.mu.Lock()
	defer w.mu.Unlock()
	var changes []StateChange
	for _, ps := range w.panes {
		status, clear := deriveStatusFromFile(ps.transcript)
		changes = append(changes, StateChange{
			PaneID:   ps.paneID,
			WindowID: ps.windowID,
			Session:  ps.session,
			Status:   status,
			Clear:    clear,
		})
	}
	return changes
}

func (w *Watcher) run() {
	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.fw.Events:
			if !ok {
				return
			}
			w.handleEvent(event.Name)
		case _, ok := <-w.fw.Errors:
			if !ok {
				return
			}
		}
	}
}

func (w *Watcher) discover() {
	panes, err := listClaudePanes()
	if err != nil {
		return
	}
	home, _ := os.UserHomeDir()
	projectsDir := filepath.Join(home, ".claude", "projects")

	w.mu.Lock()
	defer w.mu.Unlock()
	for _, p := range panes {
		encoded := encodeProjectPath(p.currentPath)
		dir := filepath.Join(projectsDir, encoded)
		transcript := latestTranscript(dir)
		if transcript == "" {
			continue
		}
		if _, exists := w.panes[transcript]; exists {
			continue
		}
		w.panes[transcript] = &paneState{
			paneID:     p.paneID,
			windowID:   p.windowID,
			session:    p.session,
			transcript: transcript,
		}
		_ = w.fw.Add(transcript)
	}
}

func (w *Watcher) handleEvent(path string) {
	w.mu.Lock()
	ps, ok := w.panes[path]
	if !ok {
		w.mu.Unlock()
		return
	}
	if ps.staleTimer != nil {
		ps.staleTimer.Stop()
	}
	staleDur := w.staleDur
	ps.staleTimer = time.AfterFunc(staleDur, func() {
		w.emit(StateChange{
			PaneID:   ps.paneID,
			WindowID: ps.windowID,
			Session:  ps.session,
			Status:   StatusStale,
		})
	})
	w.mu.Unlock()

	status, clear := deriveStatusFromFile(path)
	w.emit(StateChange{
		PaneID:   ps.paneID,
		WindowID: ps.windowID,
		Session:  ps.session,
		Status:   status,
		Clear:    clear,
	})
}

func (w *Watcher) emit(sc StateChange) {
	select {
	case w.changes <- sc:
	case <-w.done:
	}
}

type claudePane struct {
	paneID      string
	windowID    string
	session     string
	currentPath string
}

func listClaudePanes() ([]claudePane, error) {
	out, err := exec.Command("tmux", "list-panes", "-a", "-F",
		"#{pane_id}\t#{window_id}\t#{session_name}\t#{pane_current_path}\t#{pane_current_command}").Output()
	if err != nil {
		return nil, err
	}
	var panes []claudePane
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) != 5 {
			continue
		}
		if !strings.HasPrefix(parts[4], "claude") {
			continue
		}
		panes = append(panes, claudePane{
			paneID:      parts[0],
			windowID:    parts[1],
			session:     parts[2],
			currentPath: parts[3],
		})
	}
	return panes, nil
}

// encodeProjectPath maps a filesystem path to the directory name Claude Code
// uses in ~/.claude/projects/. Claude Code replaces both "/" and "." with "-".
func encodeProjectPath(path string) string {
	return strings.NewReplacer("/", "-", ".", "-").Replace(path)
}

// latestTranscript returns the most recently modified .jsonl file in dir
// that was modified within the past 24 hours.
func latestTranscript(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	var latest string
	var latestMod time.Time
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil || info.ModTime().Before(cutoff) {
			continue
		}
		if info.ModTime().After(latestMod) {
			latestMod = info.ModTime()
			latest = filepath.Join(dir, e.Name())
		}
	}
	return latest
}

type transcriptEvent struct {
	Type    string        `json:"type"`
	Message transcriptMsg `json:"message"`
}

type transcriptMsg struct {
	Role       string         `json:"role"`
	Content    []contentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
}

type contentBlock struct {
	Type string `json:"type"`
}

func deriveStatusFromFile(path string) (Status, bool) {
	events := tailTranscript(path)
	if len(events) == 0 {
		return StatusWaiting, false
	}
	return deriveStatus(events)
}

func tailTranscript(path string) []transcriptEvent {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		if t := scanner.Text(); t != "" {
			lines = append(lines, t)
		}
	}
	if len(lines) > tailN {
		lines = lines[len(lines)-tailN:]
	}
	var events []transcriptEvent
	for _, line := range lines {
		var e transcriptEvent
		if err := json.Unmarshal([]byte(line), &e); err == nil {
			events = append(events, e)
		}
	}
	return events
}

// deriveStatus walks transcript events newest-first and returns the current
// agent state. Returns (status, clear=true) when a real user message is seen,
// meaning the user already responded and the notification should be cleared.
func deriveStatus(events []transcriptEvent) (Status, bool) {
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		switch e.Type {
		case "assistant":
			if hasContentType(e.Message.Content, "tool_use") {
				return StatusRunning, false
			}
			if e.Message.StopReason == "end_turn" {
				return StatusWaiting, false
			}
			return StatusRunning, false
		case "user":
			// tool_result entries mean Claude is processing tool output — still running.
			// Any other user content (text) means the user actually responded.
			if hasContentType(e.Message.Content, "tool_result") {
				return StatusRunning, false
			}
			return "", true
		}
	}
	return StatusWaiting, false
}

func hasContentType(content []contentBlock, t string) bool {
	for _, c := range content {
		if c.Type == t {
			return true
		}
	}
	return false
}

// staleDuration reads @claude-notify-stale-minutes from tmux options,
// falling back to 5 minutes if unset or non-numeric.
func staleDuration() time.Duration {
	out, err := exec.Command("tmux", "show-option", "-gqv", "@claude-notify-stale-minutes").Output()
	if err == nil {
		if mins, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil && mins > 0 {
			return time.Duration(mins) * time.Minute
		}
	}
	return 5 * time.Minute
}
