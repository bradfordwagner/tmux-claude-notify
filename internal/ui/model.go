package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"

	"github.com/bradfordwagner/tmux-claude-notify/internal/setup"
	"github.com/bradfordwagner/tmux-claude-notify/internal/store"
	tmuxclient "github.com/bradfordwagner/tmux-claude-notify/internal/tmux"
	"github.com/bradfordwagner/tmux-claude-notify/internal/watcher"
)

var (
	accent        = lipgloss.Color("#AD8EE6")
	subtle        = lipgloss.Color("#555555")
	okColor       = lipgloss.Color("#50FA7B")
	warnColor     = lipgloss.Color("#FFB86C")
	unknColor     = lipgloss.Color("#FF5555")
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(accent)
	selectedStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4"))
	dimStyle      = lipgloss.NewStyle().Foreground(subtle)
	statusOK      = lipgloss.NewStyle().Foreground(okColor)
	statusWarn    = lipgloss.NewStyle().Foreground(warnColor)
	statusUnknown = lipgloss.NewStyle().Foreground(unknColor)

	statusStyles = map[string]lipgloss.Style{
		"waiting": lipgloss.NewStyle().Foreground(accent),
		"running": lipgloss.NewStyle().Foreground(warnColor),
		"stale":   lipgloss.NewStyle().Foreground(subtle),
	}
)

type (
	settingsChangedMsg      struct{}
	notificationsChangedMsg struct{}
)

type entry struct {
	record store.Record
	Path   string
}

type model struct {
	entries           []entry
	cursor            int
	setupResult       setup.Result
	setupMessage      string
	toast             string
	toastTimer        timer.Model
	watcher           *fsnotify.Watcher
	transcriptWatcher *watcher.Watcher
	quitting          bool
}

func newModel() (model, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return model{}, err
	}

	settingsDir := filepath.Dir(setup.SettingsPath())
	_ = fw.Add(settingsDir)

	logDir := filepath.Dir(store.LogPath())
	_ = os.MkdirAll(logDir, 0o755)
	_ = fw.Add(logDir)

	tw, err := watcher.New()
	if err != nil {
		fw.Close()
		return model{}, err
	}

	m := model{watcher: fw, transcriptWatcher: tw}
	m.setupResult, m.setupMessage = checkAndConfigure()
	if m.setupMessage != "" {
		m.toast = m.setupMessage
		m.toastTimer = timer.NewWithInterval(10*time.Second, time.Second)
	}
	m.entries = loadEntries()

	// Reconcile store against live transcript state before first render.
	for _, sc := range tw.Reconcile() {
		applyStateChange(sc)
	}
	m.entries = loadEntries()

	tw.Start()
	return m, nil
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{watchCmd(m.watcher), watcherCmd(m.transcriptWatcher)}
	if m.toast != "" {
		cmds = append(cmds, m.toastTimer.Init())
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case settingsChangedMsg:
		m.setupResult, m.setupMessage = checkAndConfigure()
		if m.setupMessage != "" {
			m.toast = m.setupMessage
			m.toastTimer = timer.NewWithInterval(10*time.Second, time.Second)
			return m, tea.Batch(watchCmd(m.watcher), m.toastTimer.Init())
		}
		return m, watchCmd(m.watcher)

	case timer.TickMsg:
		var cmd tea.Cmd
		m.toastTimer, cmd = m.toastTimer.Update(msg)
		return m, cmd

	case timer.TimeoutMsg:
		m.toastTimer, _ = m.toastTimer.Update(msg)
		m.toast = ""
		return m, nil

	case notificationsChangedMsg:
		m.entries = loadEntries()
		if m.cursor >= len(m.entries) {
			m.cursor = max(0, len(m.entries)-1)
		}
		return m, watchCmd(m.watcher)

	case watcher.StateChange:
		applyStateChange(msg)
		m.entries = loadEntries()
		if m.cursor >= len(m.entries) {
			m.cursor = max(0, len(m.entries)-1)
		}
		return m, watcherCmd(m.transcriptWatcher)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.watcher.Close()
			m.transcriptWatcher.Close()
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.entries) > 0 {
				selected := m.entries[m.cursor]
				_ = store.ClearPane(selected.record.Pane)
				_ = tmuxclient.ClearWindowStyle(selected.record.Window)
				_ = tmuxclient.ClearPopStyle(selected.record.Window)
				_ = tmuxclient.UnregisterClearHook(selected.record.Pane)
				_ = tmuxclient.SelectWindow(selected.record.Session, selected.record.Window)
				m.entries = loadEntries()
				if m.cursor >= len(m.entries) {
					m.cursor = max(0, len(m.entries)-1)
				}
				_ = tmuxclient.DetachIfShpell()
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("claude-notify") + "\n")
	if status := m.renderSetupStatus(); status != "" {
		b.WriteString(status + "\n")
	}
	if m.toast != "" {
		b.WriteString(statusOK.Render("✓ "+m.toast) + "\n")
	}
	b.WriteString("\n")

	if len(m.entries) == 0 {
		b.WriteString(dimStyle.Render("No pending notifications.") + "\n")
	} else {
		header := fmt.Sprintf("  %-12s  %-20s  %-25s  %-14s  %s",
			"STATUS", "WINDOW", "PATH", "SESSION", "AGE")
		b.WriteString(dimStyle.Render(header) + "\n")
		b.WriteString(dimStyle.Render("  " + strings.Repeat("─", 82)) + "\n")
		for i, e := range m.entries {
			r := e.record
			statusBadge := renderStatusBadge(r.Status)
			line := fmt.Sprintf("%s  %-20s  %-25s  %-14s  %s",
				statusBadge, r.WindowName, e.Path, r.Session, formatAge(r.TS))
			if i == m.cursor {
				b.WriteString(selectedStyle.Render("> ") + line + "\n")
			} else {
				b.WriteString("  " + normalStyle.Render(line) + "\n")
			}
		}
		b.WriteString("\n" + dimStyle.Render("↑/↓ or j/k to move  •  enter to switch  •  q to quit") + "\n")
	}

	return b.String()
}

var statusIcons = map[string]string{
	"waiting": "⏳",
	"running": "⚙ ",
	"stale":   "💤",
}

func renderStatusBadge(status string) string {
	if status == "" {
		status = "waiting"
	}
	style, ok := statusStyles[status]
	if !ok {
		style = dimStyle
	}
	icon := statusIcons[status]
	// Pad plain text before applying color so ANSI codes don't skew column widths.
	padded := fmt.Sprintf("%-10s", icon+" "+status)
	return style.Render(padded)
}

func (m model) renderSetupStatus() string {
	switch m.setupResult.Status {
	case setup.StatusConfigured:
		return ""
	case setup.StatusNotConfigured:
		return statusWarn.Render("⚠ hook not configured") + "\n" + dimStyle.Render("  "+m.setupResult.Message)
	default:
		return statusUnknown.Render("? " + m.setupResult.Message)
	}
}

// watchCmd blocks on the fsnotify watcher and dispatches message type by file.
func watchCmd(fw *fsnotify.Watcher) tea.Cmd {
	settingsFile := filepath.Base(setup.SettingsPath())
	logFile := filepath.Base(store.LogPath())
	return func() tea.Msg {
		for {
			select {
			case event, ok := <-fw.Events:
				if !ok {
					return nil
				}
				switch filepath.Base(event.Name) {
				case settingsFile:
					return settingsChangedMsg{}
				case logFile:
					return notificationsChangedMsg{}
				}
			case _, ok := <-fw.Errors:
				if !ok {
					return nil
				}
			}
		}
	}
}

// watcherCmd waits for the next transcript StateChange and returns it as a tea.Msg.
func watcherCmd(tw *watcher.Watcher) tea.Cmd {
	return func() tea.Msg {
		sc, ok := <-tw.Changes()
		if !ok {
			return nil
		}
		return sc
	}
}

// applyStateChange updates the JSONL store and tmux styles for a watcher event.
func applyStateChange(sc watcher.StateChange) {
	if sc.Clear {
		_ = store.ClearPane(sc.PaneID)
		_ = tmuxclient.ClearWindowStyle(sc.WindowID)
		_ = tmuxclient.ClearPopStyle(sc.WindowID)
		return
	}
	has, _ := store.HasUnclearedPane(sc.PaneID)
	if sc.Status == watcher.StatusWaiting {
		if has {
			_ = store.UpdateStatus(sc.PaneID, string(sc.Status))
		} else {
			windowName, _ := tmuxclient.WindowName(sc.PaneID)
			_ = store.Append(store.Record{
				TS:         store.NowNano(),
				Pane:       sc.PaneID,
				Window:     sc.WindowID,
				WindowName: windowName,
				Session:    sc.Session,
				Status:     string(sc.Status),
			})
		}
		_ = tmuxclient.SetWindowStyle(sc.WindowID)
	} else {
		_ = store.UpdateStatus(sc.PaneID, string(sc.Status))
	}
}

func checkAndConfigure() (setup.Result, string) {
	result := setup.Check()
	if result.Status != setup.StatusNotConfigured {
		return result, ""
	}
	msg, err := setup.Configure()
	if err != nil {
		return setup.Result{Status: setup.StatusUnknown, Message: "auto-configure failed: " + err.Error()}, ""
	}
	return setup.Result{Status: setup.StatusConfigured, Message: "Stop hook configured"}, msg
}

func loadEntries() []entry {
	records, _ := store.ReadAll()
	livePanes, _ := tmuxclient.ListLivePanes()
	liveSet := make(map[string]bool, len(livePanes))
	for _, p := range livePanes {
		liveSet[p] = true
	}
	// Deduplicate: show only the most recent uncleared record per pane.
	// Duplicates can arise from a race between the Stop hook subprocess and the
	// watcher goroutine both appending before either's HasUnclearedPane check
	// sees the other's write.
	home, _ := os.UserHomeDir()
	seen := make(map[string]bool)
	var entries []entry
	for _, r := range records { // records already sorted newest-first
		if !r.Cleared && liveSet[r.Pane] && !seen[r.Pane] {
			seen[r.Pane] = true
			p, _ := tmuxclient.PanePath(r.Pane)
			entries = append(entries, entry{record: r, Path: trimPath(p, home)})
		}
	}
	return entries
}

func trimPath(p, home string) string {
	if p == "" {
		return ""
	}
	if home != "" && strings.HasPrefix(p, home) {
		p = "~" + p[len(home):]
	}
	parts := strings.Split(strings.TrimRight(p, "/"), "/")
	if len(parts) <= 2 {
		return p
	}
	return strings.Join(parts[len(parts)-2:], "/")
}

func formatAge(tsNano int64) string {
	d := time.Since(time.Unix(0, tsNano))
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Run() error {
	m, err := newModel()
	if err != nil {
		return err
	}
	p := tea.NewProgram(m)
	_, err = p.Run()
	return err
}
