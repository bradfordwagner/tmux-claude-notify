package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
	"github.com/mattn/go-runewidth"

	"github.com/bradfordwagner/tmux-claude-notify/internal/sessions"
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
	selBg         = lipgloss.Color("#313244") // Catppuccin surface0 — row selection background
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
		"idle":    lipgloss.NewStyle().Foreground(subtle),
	}

	statusIcons = map[string]string{
		"waiting": "⏳",
		"running": "⚙ ",
		"stale":   "💤",
		"idle":    "💤",
	}
)

type viewMode int

const (
	viewNotifications viewMode = iota
	viewSessions
)

type sortField int

const (
	sortAge sortField = iota
	sortStatus
)

type (
	settingsChangedMsg      struct{}
	notificationsChangedMsg struct{}
)

// entry is a row in the Notifications view.
type entry struct {
	record         store.Record
	Path           string
	pinned         bool   // true if backed by a pinned sessions record
	sessionID      string // set when backed by sessions.jsonl (enables pin toggle)
	isSessionEntry bool   // true = came from sessions.jsonl, not notifications.jsonl
	popped         bool   // true when the pane currently has background pop active
}

// sessionEntry is a row in the Sessions view.
type sessionEntry struct {
	record   sessions.SessionRecord
	projPath string // trimmed display path
}

// projRow is a Level-1 row in the Sessions table (one per project).
type projRow struct {
	key          string // "📌 Pinned" or trimmed projPath
	count        int
	lastActivity int64
	bestStatus   string
}

type model struct {
	entries      []entry
	cursor       int
	setupResult  setup.Result
	setupMessage string
	toast        string
	toastIsError bool
	toastTimer   timer.Model
	watcher      *fsnotify.Watcher
	transcriptWatcher *watcher.Watcher
	quitting     bool
	width        int

	activeView   viewMode
	sessionItems []sessionEntry
	drillProject string // "" = projects table (Level 1); non-empty = sessions for this project (Level 2)
	sortBy       sortField
	filterActive bool
}

func (m model) termWidth() int {
	if m.width > 0 {
		return m.width
	}
	return 120
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

	_ = sessions.Compact(90 * 24 * time.Hour)

	m.entries = loadEntries()

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

func (m model) Update(msg tea.Msg) (ret tea.Model, cmd tea.Cmd) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			_ = os.WriteFile("/tmp/cn-crash.log", append([]byte(fmt.Sprintf("panic: %v\n\n", r)), buf[:n]...), 0o644)
			panic(r)
		}
	}()
	switch msg := msg.(type) {
	case settingsChangedMsg:
		m.setupResult, m.setupMessage = checkAndConfigure()
		if m.setupMessage != "" {
			m.toast = m.setupMessage
			m.toastIsError = false
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

	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case notificationsChangedMsg:
		m.entries = loadEntries()
		if m.activeView == viewSessions {
			m.sessionItems = loadSessionEntries(m.sortBy, m.filterActive)
		}
		m.clampCursor()
		return m, watchCmd(m.watcher)

	case watcher.StateChange:
		applyStateChange(msg)
		m.entries = loadEntries()
		for i, si := range m.sessionItems {
			if si.record.PaneID == msg.PaneID {
				m.sessionItems[i].record.Status = string(msg.Status)
				break
			}
		}
		m.clampCursor()
		return m, watcherCmd(m.transcriptWatcher)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.watcher.Close()
			m.transcriptWatcher.Close()
			m.quitting = true
			return m, tea.Quit

		case "esc":
			if m.activeView == viewSessions && m.drillProject != "" {
				m.drillProject = ""
				m.cursor = 0
			} else {
				m.watcher.Close()
				m.transcriptWatcher.Close()
				m.quitting = true
				return m, tea.Quit
			}

		case "tab":
			if m.activeView == viewNotifications {
				m.activeView = viewSessions
				m.cursor = 0
				m.drillProject = ""
				m.sessionItems = loadSessionEntries(m.sortBy, m.filterActive)
			} else {
				m.activeView = viewNotifications
				m.cursor = 0
				m.drillProject = ""
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			maxIdx := m.sessionListLen() - 1
			if m.activeView == viewNotifications {
				maxIdx = len(m.entries) - 1
			}
			if m.cursor < maxIdx {
				m.cursor++
			}

		case "s":
			if m.activeView == viewSessions {
				m.sortBy = (m.sortBy + 1) % 2
				m.sessionItems = loadSessionEntries(m.sortBy, m.filterActive)
				m.clampCursor()
			}

		case "f":
			if m.activeView == viewSessions {
				m.filterActive = !m.filterActive
				m.sessionItems = loadSessionEntries(m.sortBy, m.filterActive)
				m.cursor = 0
			}

		case "p":
			switch m.activeView {
			case viewSessions:
				if m.drillProject != "" {
					drilled := m.sessionsForDrill()
					if len(drilled) > 0 && m.cursor < len(drilled) {
						si := drilled[m.cursor]
						if err := sessions.SetPinned(si.record.SessionID, !si.record.Pinned); err == nil {
							m.sessionItems = loadSessionEntries(m.sortBy, m.filterActive)
							m.entries = loadEntries()
							m.clampCursor()
						}
					}
				}
			case viewNotifications:
				if len(m.entries) > 0 {
					e := m.entries[m.cursor]
					if e.sessionID != "" {
						if err := sessions.SetPinned(e.sessionID, !e.pinned); err == nil {
							m.entries = loadEntries()
							m.sessionItems = loadSessionEntries(m.sortBy, m.filterActive)
							m.clampCursor()
						}
					}
				}
			}

		case "w":
			if m.activeView == viewSessions {
				if m.drillProject == "" {
					rows := buildProjRows(m.sessionItems)
					if len(rows) > 0 && m.cursor < len(rows) {
						return m.doNewSession(m.projPathForRow(rows[m.cursor]), "neww")
					}
				} else {
					drilled := m.sessionsForDrill()
					if len(drilled) > 0 && m.cursor < len(drilled) {
						si := drilled[m.cursor]
						if si.record.PaneID != "" {
							_ = tmuxclient.SelectPane(si.record.PaneID)
							_ = tmuxclient.SelectWindow(si.record.TmuxSession, si.record.WindowID)
							_ = tmuxclient.DetachIfShpell()
						} else {
							return m.doResume(si, "neww")
						}
					}
				}
			}

		case "h":
			if m.activeView == viewSessions {
				if m.drillProject == "" {
					rows := buildProjRows(m.sessionItems)
					if len(rows) > 0 && m.cursor < len(rows) {
						return m.doNewSession(m.projPathForRow(rows[m.cursor]), "split-h")
					}
				} else {
					drilled := m.sessionsForDrill()
					if len(drilled) > 0 && m.cursor < len(drilled) {
						si := drilled[m.cursor]
						if si.record.PaneID == "" {
							return m.doResume(si, "split-h")
						}
					}
				}
			}

		case "v":
			if m.activeView == viewSessions {
				if m.drillProject == "" {
					rows := buildProjRows(m.sessionItems)
					if len(rows) > 0 && m.cursor < len(rows) {
						return m.doNewSession(m.projPathForRow(rows[m.cursor]), "split-v")
					}
				} else {
					drilled := m.sessionsForDrill()
					if len(drilled) > 0 && m.cursor < len(drilled) {
						si := drilled[m.cursor]
						if si.record.PaneID == "" {
							return m.doResume(si, "split-v")
						}
					}
				}
			}

		case "enter", " ":
			switch m.activeView {
			case viewNotifications:
				if len(m.entries) > 0 {
					selected := m.entries[m.cursor]
					if selected.isSessionEntry && selected.record.Pane == "" {
						m.toast = "No active pane — switch to Sessions tab to resume"
						m.toastIsError = false
						m.toastTimer = timer.NewWithInterval(5*time.Second, time.Second)
						return m, m.toastTimer.Init()
					}
					paneID := selected.record.Pane
					windowID := selected.record.Window
					if !selected.isSessionEntry {
						_ = tmuxclient.ClearPopStyle(paneID)
						_ = tmuxclient.UnregisterClearHook(paneID)
						_ = store.ClearPane(paneID)
						remaining, _ := store.UnclearedForWindow(windowID)
						if len(remaining) == 0 {
							_ = tmuxclient.ClearWindowStyle(windowID)
						}
					}
					_ = tmuxclient.SelectPane(paneID)
					_ = tmuxclient.SelectWindow(selected.record.Session, windowID)
					m.entries = loadEntries()
					m.clampCursor()
					_ = tmuxclient.DetachIfShpell()
				}
			case viewSessions:
				if m.drillProject == "" {
					rows := buildProjRows(m.sessionItems)
					if len(rows) > 0 && m.cursor < len(rows) {
						m.drillProject = rows[m.cursor].key
						m.cursor = 0
					}
				} else {
					drilled := m.sessionsForDrill()
					if len(drilled) > 0 && m.cursor < len(drilled) {
						si := drilled[m.cursor]
						if si.record.PaneID != "" {
							_ = tmuxclient.SelectPane(si.record.PaneID)
							_ = tmuxclient.SelectWindow(si.record.TmuxSession, si.record.WindowID)
							_ = tmuxclient.DetachIfShpell()
						} else {
							return m.doResume(si, "neww")
						}
					}
				}
			}
		}
	}
	return m, nil
}

// doResume opens a closed session. mode is one of "neww", "split-h", "split-v".
func (m model) doResume(si sessionEntry, mode string) (tea.Model, tea.Cmd) {
	if !tmuxclient.InTmux() {
		m.toast = "Cannot resume: not in tmux"
		m.toastIsError = true
		m.toastTimer = timer.NewWithInterval(5*time.Second, time.Second)
		return m, m.toastTimer.Init()
	}
	projPath := sessions.RecoverPath(si.record.EncodedPath, si.record.ProjectPath)
	if projPath == "" {
		m.toast = "Cannot resume: project path unknown"
		m.toastIsError = true
		m.toastTimer = timer.NewWithInterval(5*time.Second, time.Second)
		return m, m.toastTimer.Init()
	}
	outer := tmuxclient.OuterSession()
	var args []string
	switch mode {
	case "split-h", "split-v":
		flag := "-h"
		if mode == "split-v" {
			flag = "-v"
		}
		args = []string{"split-window", flag, "-c", projPath}
		if outer != "" {
			args = append(args, "-t", outer)
		}
		args = append(args, "--", "claude", "--resume", si.record.SessionID)
	default: // "neww"
		args = []string{"neww", "-c", projPath}
		if outer != "" {
			args = append(args, "-t", outer)
		}
		args = append(args, "--", "claude", "--resume", si.record.SessionID)
	}
	_ = exec.Command("tmux", args...).Run()
	_ = tmuxclient.DetachIfShpell()
	return m, nil
}

// doNewSession opens a fresh claude session (no --resume) in projPath.
// mode is "neww", "split-h", or "split-v". neww names the window after the leaf dir.
func (m model) doNewSession(projPath, mode string) (tea.Model, tea.Cmd) {
	if !tmuxclient.InTmux() {
		m.toast = "Cannot open: not in tmux"
		m.toastIsError = true
		m.toastTimer = timer.NewWithInterval(5*time.Second, time.Second)
		return m, m.toastTimer.Init()
	}
	if projPath == "" {
		m.toast = "Cannot open: project path unknown"
		m.toastIsError = true
		m.toastTimer = timer.NewWithInterval(5*time.Second, time.Second)
		return m, m.toastTimer.Init()
	}
	outer := tmuxclient.OuterSession()
	var args []string
	switch mode {
	case "split-h", "split-v":
		flag := "-h"
		if mode == "split-v" {
			flag = "-v"
		}
		args = []string{"split-window", flag, "-c", projPath}
		if outer != "" {
			args = append(args, "-t", outer)
		}
		args = append(args, "--", "claude")
	default: // "neww"
		leafName := filepath.Base(projPath)
		args = []string{"neww", "-c", projPath, "-n", leafName}
		if outer != "" {
			args = append(args, "-t", outer)
		}
		args = append(args, "--", "claude")
	}
	// Run synchronously — tmux neww/split-window returns as soon as the window
	// is created; DetachIfShpell must fire after, not before.
	_ = exec.Command("tmux", args...).Run()
	_ = tmuxclient.DetachIfShpell()
	return m, nil
}

// projPathForRow finds the real filesystem path for a Level-1 project row by
// recovering the path from any session in that group.
func (m model) projPathForRow(row projRow) string {
	for _, e := range m.sessionItems {
		if groupKeyFor(e) == row.key {
			path := sessions.RecoverPath(e.record.EncodedPath, e.record.ProjectPath)
			if path != "" {
				return path
			}
		}
	}
	return ""
}

func (m *model) clampCursor() {
	listLen := m.sessionListLen()
	if m.activeView == viewNotifications {
		listLen = len(m.entries)
	}
	if m.cursor >= listLen {
		m.cursor = max(0, listLen-1)
	}
}

func (m model) sessionListLen() int {
	if m.drillProject != "" {
		return len(m.sessionsForDrill())
	}
	return len(buildProjRows(m.sessionItems))
}

func (m model) sessionsForDrill() []sessionEntry {
	var result []sessionEntry
	for _, e := range m.sessionItems {
		if groupKeyFor(e) == m.drillProject {
			result = append(result, e)
		}
	}
	return result
}

func groupKeyFor(e sessionEntry) string {
	if e.record.Pinned {
		return "📌 Pinned"
	}
	if e.projPath != "" {
		return e.projPath
	}
	return "(unknown)"
}

// buildProjRows computes Level-1 project rows from the flat session list.
func buildProjRows(items []sessionEntry) []projRow {
	var order []string
	type acc struct {
		count        int
		lastActivity int64
		bestStatus   string
	}
	seen := make(map[string]*acc)
	for _, e := range items {
		key := groupKeyFor(e)
		if _, ok := seen[key]; !ok {
			seen[key] = &acc{bestStatus: "idle"}
			order = append(order, key)
		}
		a := seen[key]
		a.count++
		if e.record.LastActivity > a.lastActivity {
			a.lastActivity = e.record.LastActivity
		}
		if statusPriority(e.record.Status) < statusPriority(a.bestStatus) {
			a.bestStatus = e.record.Status
		}
	}
	rows := make([]projRow, 0, len(order))
	for _, k := range order {
		a := seen[k]
		rows = append(rows, projRow{key: k, count: a.count, lastActivity: a.lastActivity, bestStatus: a.bestStatus})
	}
	return rows
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	var b strings.Builder
	b.WriteString(m.renderTabHeader() + "\n")
	if m.activeView == viewNotifications {
		if status := m.renderSetupStatus(); status != "" {
			b.WriteString(status + "\n")
		}
	}
	if m.toast != "" {
		style := statusOK
		prefix := "✓ "
		if m.toastIsError {
			style = statusWarn
			prefix = "⚠ "
		}
		b.WriteString(style.Render(prefix+m.toast) + "\n")
	}
	b.WriteString("\n")
	switch m.activeView {
	case viewNotifications:
		b.WriteString(m.renderNotificationsView())
	case viewSessions:
		b.WriteString(m.renderSessionsView())
	}
	return b.String()
}

func (m model) renderTabHeader() string {
	notifLabel := " Notifications "
	sessLabel := " Sessions "
	switch m.activeView {
	case viewNotifications:
		active := titleStyle.Render("[" + notifLabel + "]")
		inactive := dimStyle.Render("  " + strings.TrimSpace(sessLabel) + "  ")
		return active + inactive
	default: // viewSessions
		inactive := dimStyle.Render("  " + strings.TrimSpace(notifLabel) + "  ")
		active := titleStyle.Render("[" + sessLabel + "]")
		sortNames := []string{"age", "status"}
		extra := dimStyle.Render("   sort: " + sortNames[m.sortBy])
		if m.filterActive {
			extra += dimStyle.Render("  filter: active panes")
		}
		if m.drillProject != "" {
			extra += dimStyle.Render("  esc: back")
		}
		return inactive + active + extra
	}
}

func (m model) renderNotificationsView() string {
	var b strings.Builder
	if len(m.entries) == 0 {
		b.WriteString(dimStyle.Render("No pending notifications.") + "\n")
	} else {
		// Fixed overhead: 2 prefix + 12 status + 2 + 3 pin + 2 + 1 pop + 2 + 20 window + 2 + pathWidth + 2 + 14 session + 2 + 10 age = 74
		pathWidth := max(10, m.termWidth()-74)
		header := fmt.Sprintf("  %-12s  %-3s  %s  %-20s  %-*s  %-14s  %s",
			"STATUS", "PIN", "P", "WINDOW", pathWidth, "PATH", "SESSION", "AGE")
		b.WriteString(dimStyle.Render(header) + "\n")
		b.WriteString(dimStyle.Render("  "+strings.Repeat("─", m.termWidth()-2)) + "\n")
		for i, e := range m.entries {
			r := e.record
			pinCol := "   "
			if e.pinned {
				pinCol = "📌 "
			}
			path := runewidth.Truncate(e.Path, pathWidth, "…")
			path = runewidth.FillRight(path, pathWidth)
			if i == m.cursor {
				bgs := lipgloss.NewStyle().Background(selBg)
				sep := bgs.Render("  ")
				badge := renderStatusBadge(r.Status, selBg)
				pin := bgs.Render(pinCol)
				pop := bgs.Render(" ")
				if e.popped {
					pop = lipgloss.NewStyle().Foreground(accent).Background(selBg).Render("●")
				}
				win := bgs.Render(fmt.Sprintf("%-20s", r.WindowName))
				ps := bgs.Render(path)
				sess := bgs.Render(fmt.Sprintf("%-14s", r.Session))
				age := bgs.Render(formatAge(r.TS))
				ind := selectedStyle.Background(selBg).Render("> ")
				innerW := 2 + 12 + 2 + runewidth.StringWidth(pinCol) + 2 + 1 + 2 + 20 + 2 + pathWidth + 2 + 14 + 2 + runewidth.StringWidth(formatAge(r.TS))
				pad := bgs.Render(strings.Repeat(" ", max(0, m.termWidth()-innerW)))
				b.WriteString(ind+badge+sep+pin+sep+pop+sep+win+sep+ps+sep+sess+sep+age+pad+"\n")
			} else {
				popDisp := " "
				if e.popped {
					popDisp = lipgloss.NewStyle().Foreground(accent).Render("●")
				}
				line := fmt.Sprintf("%s  %s  %s  %-20s  %s  %-14s  %s",
					renderStatusBadge(r.Status), pinCol, popDisp, r.WindowName, path, r.Session, formatAge(r.TS))
				b.WriteString("  " + normalStyle.Render(line) + "\n")
			}
		}
		b.WriteString("\n" + dimStyle.Render("↑/↓ j/k  •  enter: focus  •  p: pin  •  tab: Sessions  •  q: quit") + "\n")
	}
	return b.String()
}

func (m model) renderSessionsView() string {
	var b strings.Builder

	if m.drillProject == "" {
		// ── Level 1: projects table ──────────────────────────────────────────
		rows := buildProjRows(m.sessionItems)
		if len(rows) == 0 {
			msg := "No sessions discovered."
			if m.filterActive {
				msg = "No active sessions."
			}
			b.WriteString(dimStyle.Render(msg) + "\n")
			return b.String()
		}
		// Fixed overhead: 2 prefix + 12 status + 2 + 2 + 5 count + 2 + 10 age = 35
		projWidth := max(15, m.termWidth()-35)
		header := fmt.Sprintf("  %-12s  %-*s  %5s  %s", "STATUS", projWidth, "PROJECT", "COUNT", "LAST USED")
		b.WriteString(dimStyle.Render(header) + "\n")
		b.WriteString(dimStyle.Render("  "+strings.Repeat("─", m.termWidth()-2)) + "\n")
		for i, row := range rows {
			status := row.bestStatus
			if status == "" {
				status = "idle"
			}
			proj := runewidth.Truncate(row.key, projWidth, "…")
			proj = runewidth.FillRight(proj, projWidth)
			if i == m.cursor {
				bgs := lipgloss.NewStyle().Background(selBg)
				sep := bgs.Render("  ")
				badge := renderStatusBadge(status, selBg)
				prj := bgs.Render(proj)
				cnt := bgs.Render(fmt.Sprintf("%5d", row.count))
				age := bgs.Render(formatAge(row.lastActivity))
				ind := selectedStyle.Background(selBg).Render("> ")
				innerW := 2 + 12 + 2 + projWidth + 2 + 5 + 2 + runewidth.StringWidth(formatAge(row.lastActivity))
				pad := bgs.Render(strings.Repeat(" ", max(0, m.termWidth()-innerW)))
				b.WriteString(ind+badge+sep+prj+sep+cnt+sep+age+pad+"\n")
			} else {
				badge := renderStatusBadge(status)
				line := fmt.Sprintf("%s  %s  %5d  %s", badge, proj, row.count, formatAge(row.lastActivity))
				b.WriteString("  " + normalStyle.Render(line) + "\n")
			}
		}
		b.WriteString("\n" + dimStyle.Render("↑/↓ j/k  •  enter: sessions  •  w: new win  h: split-h  v: split-v  •  s: sort  •  f: filter  •  tab: Notifications  •  q: quit") + "\n")
		return b.String()
	}

	// ── Level 2: sessions for drillProject ───────────────────────────────────
	b.WriteString(titleStyle.Render("  ← "+m.drillProject) + "\n")
	b.WriteString(dimStyle.Render("  "+strings.Repeat("─", min(runewidth.StringWidth(m.drillProject)+4, 50))) + "\n")

	drilled := m.sessionsForDrill()
	if len(drilled) == 0 {
		b.WriteString(dimStyle.Render("  No sessions.") + "\n")
	} else {
		header := fmt.Sprintf("  %-12s  %-3s  %-10s  %s", "STATUS", "PIN", "SESSION", "AGE")
		b.WriteString(dimStyle.Render(header) + "\n")
		b.WriteString(dimStyle.Render("  "+strings.Repeat("─", 44)) + "\n")
		for i, e := range drilled {
			r := e.record
			status := r.Status
			if status == "" {
				status = "idle"
			}
			pinCol := "   "
			if r.Pinned {
				pinCol = "📌 "
			}
			sid := r.SessionID
			if len(sid) > 8 {
				sid = sid[:8]
			}
			if i == m.cursor {
				bgs := lipgloss.NewStyle().Background(selBg)
				sep := bgs.Render("  ")
				badge := renderStatusBadge(status, selBg)
				pin := bgs.Render(pinCol)
				sidStr := bgs.Render(fmt.Sprintf("%-10s", sid))
				age := bgs.Render(formatAge(r.LastActivity))
				ind := selectedStyle.Background(selBg).Render("> ")
				innerW := 2 + 12 + 2 + runewidth.StringWidth(pinCol) + 2 + 10 + 2 + runewidth.StringWidth(formatAge(r.LastActivity))
				pad := bgs.Render(strings.Repeat(" ", max(0, m.termWidth()-innerW)))
				b.WriteString(ind+badge+sep+pin+sep+sidStr+sep+age+pad+"\n")
			} else {
				badge := renderStatusBadge(status)
				line := fmt.Sprintf("%s  %s  %-10s  %s", badge, pinCol, sid, formatAge(r.LastActivity))
				b.WriteString("  " + normalStyle.Render(line) + "\n")
			}
		}
	}
	b.WriteString("\n" + dimStyle.Render("↑/↓ j/k  •  enter/w: new win  h: split-h  v: split-v  •  p: pin  •  esc: back  •  tab: Notifications  •  q: quit") + "\n")
	return b.String()
}

// renderStatusBadge renders a status string with its icon, padded to a fixed
// visual width using runewidth so emoji-width differences don't misalign columns.
// Pass an optional background color to bake it into the badge (needed for full-width row highlights).
func renderStatusBadge(status string, bg ...lipgloss.Color) string {
	if status == "" {
		status = "waiting"
	}
	style, ok := statusStyles[status]
	if !ok {
		style = dimStyle
	}
	if len(bg) > 0 {
		style = style.Background(bg[0])
	}
	icon, ok := statusIcons[status]
	if !ok {
		icon = "  "
	}
	text := icon + " " + status
	const targetWidth = 12
	w := runewidth.StringWidth(text)
	if w < targetWidth {
		text += strings.Repeat(" ", targetWidth-w)
	}
	return style.Render(text)
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

func watcherCmd(tw *watcher.Watcher) tea.Cmd {
	return func() tea.Msg {
		sc, ok := <-tw.Changes()
		if !ok {
			return nil
		}
		return sc
	}
}

func applyStateChange(sc watcher.StateChange) {
	if sc.Clear {
		_ = tmuxclient.ClearPopStyle(sc.PaneID)
		_ = store.ClearPane(sc.PaneID)
		remaining, _ := store.UnclearedForWindow(sc.WindowID)
		if len(remaining) == 0 {
			_ = tmuxclient.ClearWindowStyle(sc.WindowID)
		}
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

	sessRecords, _ := sessions.ReadAll()
	paneToSession := make(map[string]sessions.SessionRecord)
	for _, sr := range sessRecords {
		if sr.PaneID != "" {
			paneToSession[sr.PaneID] = sr
		}
	}

	home, _ := os.UserHomeDir()
	seenPane := make(map[string]bool)
	seenSession := make(map[string]bool)

	var notifPinned []entry
	var notifNormal []entry
	for _, r := range records {
		if r.Cleared || seenPane[r.Pane] {
			continue
		}
		seenPane[r.Pane] = true
		p, _ := tmuxclient.PanePath(r.Pane)
		path := trimPath(p, home)
		if path == "" && !liveSet[r.Pane] {
			path = "(gone)"
		}
		pinned := false
		sid := ""
		if sr, ok := paneToSession[r.Pane]; ok {
			pinned = sr.Pinned
			sid = sr.SessionID
			seenSession[sr.SessionID] = true
		}
		e := entry{record: r, Path: path, pinned: pinned, sessionID: sid, popped: tmuxclient.IsPanePopped(r.Pane)}
		if pinned {
			notifPinned = append(notifPinned, e)
		} else {
			notifNormal = append(notifNormal, e)
		}
	}

	var sessActivePinned []entry
	var sessActiveNormal []entry
	var sessIdlePinned []entry

	for _, sr := range sessRecords {
		if seenSession[sr.SessionID] {
			continue
		}
		if sr.PaneID != "" && seenPane[sr.PaneID] {
			continue
		}
		if sr.PaneID == "" && !sr.Pinned {
			continue
		}

		seenSession[sr.SessionID] = true
		if sr.PaneID != "" {
			seenPane[sr.PaneID] = true
		}

		status := sr.Status
		if status == "" {
			status = "idle"
		}
		path := trimPath(sr.ProjectPath, home)
		if path == "" {
			recovered := sessions.RecoverPath(sr.EncodedPath, "")
			path = trimPath(recovered, home)
		}
		if path == "" {
			path = "(unknown)"
		}

		e := entry{
			record: store.Record{
				TS:         sr.LastActivity,
				Pane:       sr.PaneID,
				Window:     sr.WindowID,
				WindowName: sr.WindowName,
				Session:    sr.TmuxSession,
				Status:     status,
			},
			Path:           path,
			pinned:         sr.Pinned,
			sessionID:      sr.SessionID,
			isSessionEntry: true,
		}

		switch {
		case sr.Pinned && sr.PaneID != "":
			sessActivePinned = append(sessActivePinned, e)
		case sr.Pinned && sr.PaneID == "":
			sessIdlePinned = append(sessIdlePinned, e)
		default:
			sessActiveNormal = append(sessActiveNormal, e)
		}
	}

	var result []entry
	result = append(result, notifPinned...)
	result = append(result, sessActivePinned...)
	result = append(result, sessIdlePinned...)
	result = append(result, notifNormal...)
	result = append(result, sessActiveNormal...)

	// Sort: pinned first, then popped (active background highlight) first, then most recent.
	sort.SliceStable(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.pinned != b.pinned {
			return a.pinned
		}
		if a.popped != b.popped {
			return a.popped
		}
		return a.record.TS > b.record.TS
	})
	return result
}

func loadSessionEntries(by sortField, filterActive bool) []sessionEntry {
	known, _ := sessions.ReadAll()

	discovered, _ := sessions.DiscoverAll()
	knownIDs := make(map[string]bool, len(known))
	for _, r := range known {
		knownIDs[r.SessionID] = true
	}
	for _, d := range discovered {
		if !knownIDs[d.SessionID] {
			known = append(known, d)
		}
	}

	if filterActive {
		var filtered []sessions.SessionRecord
		for _, r := range known {
			if r.PaneID != "" || r.Pinned {
				filtered = append(filtered, r)
			}
		}
		known = filtered
	}

	home, _ := os.UserHomeDir()

	entries := make([]sessionEntry, 0, len(known))
	for _, r := range known {
		projPath := sessions.RecoverPath(r.EncodedPath, r.ProjectPath)
		trimmed := trimPath(projPath, home)
		if trimmed == "" && projPath != "" {
			trimmed = projPath
		}
		entries = append(entries, sessionEntry{record: r, projPath: trimmed})
	}

	less := func(a, b sessionEntry) bool {
		switch by {
		case sortStatus:
			return statusPriority(a.record.Status) < statusPriority(b.record.Status)
		default:
			return a.record.LastActivity > b.record.LastActivity
		}
	}

	var pinned []sessionEntry
	unpinnedByGroup := make(map[string][]sessionEntry)
	var groupOrder []string
	seen := make(map[string]bool)
	for _, e := range entries {
		if e.record.Pinned {
			pinned = append(pinned, e)
		} else {
			key := e.projPath
			if key == "" {
				key = "(unknown)"
			}
			if !seen[key] {
				groupOrder = append(groupOrder, key)
				seen[key] = true
			}
			unpinnedByGroup[key] = append(unpinnedByGroup[key], e)
		}
	}

	sort.SliceStable(pinned, func(i, j int) bool { return less(pinned[i], pinned[j]) })

	groupMaxAge := make(map[string]int64, len(groupOrder))
	for k := range unpinnedByGroup {
		g := unpinnedByGroup[k]
		sort.SliceStable(g, func(i, j int) bool { return less(g[i], g[j]) })
		unpinnedByGroup[k] = g
		for _, e := range g {
			if e.record.LastActivity > groupMaxAge[k] {
				groupMaxAge[k] = e.record.LastActivity
			}
		}
	}

	if by == sortAge {
		sort.SliceStable(groupOrder, func(i, j int) bool {
			return groupMaxAge[groupOrder[i]] > groupMaxAge[groupOrder[j]]
		})
	} else {
		sort.Strings(groupOrder)
	}

	result := make([]sessionEntry, 0, len(entries))
	result = append(result, pinned...)
	for _, k := range groupOrder {
		result = append(result, unpinnedByGroup[k]...)
	}
	return result
}

func statusPriority(status string) int {
	switch status {
	case "waiting":
		return 0
	case "running":
		return 1
	case "stale":
		return 2
	default:
		return 3
	}
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

func Run() error {
	m, err := newModel()
	if err != nil {
		return err
	}
	p := tea.NewProgram(m)
	_, err = p.Run()
	return err
}
