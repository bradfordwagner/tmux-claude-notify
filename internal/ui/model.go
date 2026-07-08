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

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/timer"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
	"github.com/mattn/go-runewidth"
	"github.com/sahilm/fuzzy"

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
	selBg         = lipgloss.Color("#313244")
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

type entry struct {
	record         store.Record
	Path           string
	pinned         bool
	sessionID      string
	isSessionEntry bool
	popped         bool
}

type sessionEntry struct {
	record   sessions.SessionRecord
	projPath string
}

type projRow struct {
	key          string
	count        int
	lastActivity int64
	bestStatus   string
}

type model struct {
	entries           []entry
	cursor            int
	setupResult       setup.Result
	setupMessage      string
	toast             string
	toastIsError      bool
	toastTimer        timer.Model
	watcher           *fsnotify.Watcher
	transcriptWatcher *watcher.Watcher
	quitting          bool
	width             int
	height            int
	vp                viewport.Model

	activeView   viewMode
	sessionItems []sessionEntry
	drillProject string
	sortBy       sortField
	filterActive bool

	searchMode  bool
	searchFocus bool
	searchQuery string
	searchInput textinput.Model
}

func (m model) termWidth() int {
	if m.width > 0 {
		return m.width
	}
	return 120
}

func (m model) termHeight() int {
	if m.height > 0 {
		return m.height
	}
	return 24
}

// countLines returns the number of rendered lines in s, or 0 for empty strings.
func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(strings.TrimRight(s, "\n"), "\n") + 1
}

// fixedRowCount returns the number of terminal lines consumed by elements
// outside the scrollable viewport (header area + footer).
// This must stay in sync with View() — add here whenever a new fixed element is added.
func (m model) fixedRowCount() int {
	n := 1 // tab header
	if m.searchMode {
		n++
	}
	if m.activeView == viewNotifications {
		n += countLines(m.renderSetupStatus())
	}
	if m.toast != "" {
		n++
	}
	n += 2 // blank separator + footer hint
	return n
}

// recalcViewport resizes the viewport to fill available height/width and
// refreshes its content. Call whenever terminal size or fixed-element
// visibility changes, or when data reloads.
func (m *model) recalcViewport() {
	m.vp.Width = m.termWidth()
	m.vp.Height = max(1, m.termHeight()-m.fixedRowCount())
	m.vp.SetContent(m.renderListContent())
}

// contentHeaderLines returns the number of non-data lines rendered before
// the first selectable row in the current view's content string. Used by
// ensureCursorVisible to map cursor index → content line number.
func (m model) contentHeaderLines() int {
	switch m.activeView {
	case viewNotifications:
		if len(m.filteredEntries()) > 0 {
			return 2 // column header + separator
		}
		return 0
	case viewSessions:
		if m.drillProject != "" {
			if len(m.sessionsForDrill()) > 0 {
				return 4 // title + sep + column header + sep
			}
			return 2 // title + sep before "No sessions" message
		}
		if len(buildProjRows(m.filteredSessions())) > 0 {
			return 2 // column header + separator
		}
		return 0
	}
	return 0
}

// ensureCursorVisible adjusts the viewport's Y offset so the selected row
// is always within the visible window (minimum-scroll invariant).
func (m *model) ensureCursorVisible() {
	line := m.cursor + m.contentHeaderLines()
	if line < m.vp.YOffset {
		m.vp.SetYOffset(line)
	} else if line >= m.vp.YOffset+m.vp.Height {
		m.vp.SetYOffset(line - m.vp.Height + 1)
	}
}

// renderListContent returns the full scrollable content string for the
// current view and drill level. This is passed to viewport.SetContent.
func (m model) renderListContent() string {
	switch m.activeView {
	case viewNotifications:
		return m.renderNotificationsContent()
	case viewSessions:
		if m.drillProject != "" {
			return m.renderSessionsL2Content()
		}
		return m.renderSessionsL1Content()
	}
	return ""
}

// footerHint returns the total selectable item count and the hint text for
// the current view. Used by renderFooter to compute scroll indicator placement.
func (m model) footerHint() (total int, hint string) {
	switch m.activeView {
	case viewNotifications:
		return len(m.filteredEntries()), "↑/↓ j/k  •  enter: focus  •  p: pin  •  tab: Sessions  •  q: quit"
	case viewSessions:
		if m.drillProject != "" {
			return len(m.filteredDrill()), "↑/↓ j/k  •  enter/w: new win  h: split-h  v: split-v  •  p: pin  •  esc: back  •  tab: Notifications  •  q: quit"
		}
		return len(buildProjRows(m.filteredSessions())), "↑/↓ j/k  •  enter: sessions  •  w: new win  h: split-h  v: split-v  •  s: sort  •  f: filter  •  tab: Notifications  •  q: quit"
	}
	return 0, ""
}

// renderFooter renders the footer hint line. When the list overflows the
// viewport, a right-aligned position indicator "cursor+1/total" is appended.
func (m model) renderFooter() string {
	total, hint := m.footerHint()
	hintStr := dimStyle.Render(hint)
	if total <= m.vp.Height {
		return hintStr
	}
	indicator := fmt.Sprintf("%d/%d", m.cursor+1, total)
	hintW := runewidth.StringWidth(hint)
	indicW := runewidth.StringWidth(indicator)
	gap := m.termWidth() - hintW - indicW - 2
	if gap < 1 {
		gap = 1
	}
	return hintStr + strings.Repeat(" ", gap) + dimStyle.Render(indicator)
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

	si := textinput.New()
	si.Placeholder = "fuzzy filter…"
	si.CharLimit = 80

	m := model{watcher: fw, transcriptWatcher: tw, searchInput: si}
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

	// Initialize viewport with safe defaults; resized on first WindowSizeMsg.
	m.vp = viewport.New(m.termWidth(), max(1, m.termHeight()-m.fixedRowCount()))
	m.vp.SetContent(m.renderListContent())

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
			m.recalcViewport()
			return m, tea.Batch(watchCmd(m.watcher), m.toastTimer.Init())
		}
		m.recalcViewport()
		return m, watchCmd(m.watcher)

	case timer.TickMsg:
		var cmd tea.Cmd
		m.toastTimer, cmd = m.toastTimer.Update(msg)
		return m, cmd

	case timer.TimeoutMsg:
		m.toastTimer, _ = m.toastTimer.Update(msg)
		m.toast = ""
		m.recalcViewport()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcViewport()
		m.ensureCursorVisible()
		return m, nil

	case notificationsChangedMsg:
		m.entries = loadEntries()
		if m.activeView == viewSessions {
			m.sessionItems = loadSessionEntries(m.sortBy, m.filterActive)
		}
		m.clampCursor()
		m.recalcViewport()
		m.ensureCursorVisible()
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
		m.recalcViewport()
		m.ensureCursorVisible()
		return m, watcherCmd(m.transcriptWatcher)

	case tea.KeyMsg:
		// Input-focused search mode: alphanumeric keys go to the textinput.
		if m.searchMode && m.searchFocus {
			switch msg.String() {
			case "ctrl+c":
				m.watcher.Close()
				m.transcriptWatcher.Close()
				m.quitting = true
				return m, tea.Quit

			case "esc":
				m.clearSearch()
				m.recalcViewport()
				return m, nil

			case "tab":
				m.searchFocus = false
				m.searchInput.Blur()
				return m, nil

			case "enter", " ":
				// Fall through to normal enter handler below.

			default:
				var tiCmd tea.Cmd
				m.searchInput, tiCmd = m.searchInput.Update(msg)
				m.searchQuery = m.searchInput.Value()
				m.clampCursorToFiltered()
				m.recalcViewport()
				m.ensureCursorVisible()
				return m, tiCmd
			}
		}

		switch msg.String() {
		case "q", "ctrl+c":
			m.watcher.Close()
			m.transcriptWatcher.Close()
			m.quitting = true
			return m, tea.Quit

		case "/":
			if !m.searchMode {
				m.searchMode = true
				m.searchFocus = true
				m.searchInput.Focus()
				m.recalcViewport()
			}

		case "esc":
			if m.searchMode {
				m.clearSearch()
				m.recalcViewport()
			} else if m.activeView == viewSessions && m.drillProject != "" {
				m.drillProject = ""
				m.cursor = 0
				m.recalcViewport()
			} else {
				m.watcher.Close()
				m.transcriptWatcher.Close()
				m.quitting = true
				return m, tea.Quit
			}

		case "tab":
			if m.searchMode {
				m.searchFocus = true
				m.searchInput.Focus()
			} else {
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
				m.recalcViewport()
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			m.vp.SetContent(m.renderListContent())
			m.ensureCursorVisible()

		case "down", "j":
			maxIdx := m.filteredListLen() - 1
			if m.cursor < maxIdx {
				m.cursor++
			}
			m.vp.SetContent(m.renderListContent())
			m.ensureCursorVisible()

		case "ctrl+d":
			half := max(1, m.vp.Height/2)
			total := m.filteredListLen()
			m.cursor = min(m.cursor+half, max(0, total-1))
			m.vp.SetContent(m.renderListContent())
			m.ensureCursorVisible()

		case "ctrl+u":
			half := max(1, m.vp.Height/2)
			m.cursor = max(0, m.cursor-half)
			m.vp.SetContent(m.renderListContent())
			m.ensureCursorVisible()

		case "pgdown":
			total := m.filteredListLen()
			m.cursor = min(m.cursor+m.vp.Height, max(0, total-1))
			m.vp.SetContent(m.renderListContent())
			m.ensureCursorVisible()

		case "pgup":
			m.cursor = max(0, m.cursor-m.vp.Height)
			m.vp.SetContent(m.renderListContent())
			m.ensureCursorVisible()

		case "s":
			if m.activeView == viewSessions {
				m.sortBy = (m.sortBy + 1) % 2
				m.sessionItems = loadSessionEntries(m.sortBy, m.filterActive)
				m.clampCursor()
				m.recalcViewport()
				m.ensureCursorVisible()
			}

		case "f":
			if m.activeView == viewSessions {
				m.filterActive = !m.filterActive
				m.sessionItems = loadSessionEntries(m.sortBy, m.filterActive)
				m.cursor = 0
				m.recalcViewport()
			}

		case "p":
			switch m.activeView {
			case viewSessions:
				if m.drillProject != "" {
					drilled := m.filteredDrill()
					if len(drilled) > 0 && m.cursor < len(drilled) {
						si := drilled[m.cursor]
						if err := sessions.SetPinned(si.record.SessionID, !si.record.Pinned); err == nil {
							m.sessionItems = loadSessionEntries(m.sortBy, m.filterActive)
							m.entries = loadEntries()
							m.clampCursor()
							m.recalcViewport()
							m.ensureCursorVisible()
						}
					}
				}
			case viewNotifications:
				filtered := m.filteredEntries()
				if len(filtered) > 0 && m.cursor < len(filtered) {
					e := filtered[m.cursor]
					if e.sessionID != "" {
						if err := sessions.SetPinned(e.sessionID, !e.pinned); err == nil {
							m.entries = loadEntries()
							m.sessionItems = loadSessionEntries(m.sortBy, m.filterActive)
							m.clampCursor()
							m.recalcViewport()
							m.ensureCursorVisible()
						}
					}
				}
			}

		case "w":
			if m.activeView == viewSessions {
				if m.drillProject == "" {
					rows := buildProjRows(m.filteredSessions())
					if len(rows) > 0 && m.cursor < len(rows) {
						return m.doNewSession(m.projPathForRow(rows[m.cursor]), "neww")
					}
				} else {
					drilled := m.filteredDrill()
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
					rows := buildProjRows(m.filteredSessions())
					if len(rows) > 0 && m.cursor < len(rows) {
						return m.doNewSession(m.projPathForRow(rows[m.cursor]), "split-h")
					}
				} else {
					drilled := m.filteredDrill()
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
					rows := buildProjRows(m.filteredSessions())
					if len(rows) > 0 && m.cursor < len(rows) {
						return m.doNewSession(m.projPathForRow(rows[m.cursor]), "split-v")
					}
				} else {
					drilled := m.filteredDrill()
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
				filtered := m.filteredEntries()
				if len(filtered) > 0 && m.cursor < len(filtered) {
					selected := filtered[m.cursor]
					if selected.isSessionEntry && selected.record.Pane == "" {
						m.toast = "No active pane — switch to Sessions tab to resume"
						m.toastIsError = false
						m.toastTimer = timer.NewWithInterval(5*time.Second, time.Second)
						m.recalcViewport()
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
					rows := buildProjRows(m.filteredSessions())
					if len(rows) > 0 && m.cursor < len(rows) {
						m.drillProject = rows[m.cursor].key
						m.clearSearch()
						m.cursor = 0
						m.recalcViewport()
					}
				} else {
					drilled := m.filteredDrill()
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

func (m *model) clearSearch() {
	m.searchMode = false
	m.searchFocus = false
	m.searchQuery = ""
	m.searchInput.SetValue("")
	m.searchInput.Blur()
}

func (m model) doResume(si sessionEntry, mode string) (tea.Model, tea.Cmd) {
	if !tmuxclient.InTmux() {
		m.toast = "Cannot resume: not in tmux"
		m.toastIsError = true
		m.toastTimer = timer.NewWithInterval(5*time.Second, time.Second)
		m.recalcViewport()
		return m, m.toastTimer.Init()
	}
	projPath := sessions.RecoverPath(si.record.EncodedPath, si.record.ProjectPath)
	if projPath == "" {
		m.toast = "Cannot resume: project path unknown"
		m.toastIsError = true
		m.toastTimer = timer.NewWithInterval(5*time.Second, time.Second)
		m.recalcViewport()
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
	default:
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

func (m model) doNewSession(projPath, mode string) (tea.Model, tea.Cmd) {
	if !tmuxclient.InTmux() {
		m.toast = "Cannot open: not in tmux"
		m.toastIsError = true
		m.toastTimer = timer.NewWithInterval(5*time.Second, time.Second)
		m.recalcViewport()
		return m, m.toastTimer.Init()
	}
	if projPath == "" {
		m.toast = "Cannot open: project path unknown"
		m.toastIsError = true
		m.toastTimer = timer.NewWithInterval(5*time.Second, time.Second)
		m.recalcViewport()
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
	default:
		leafName := filepath.Base(projPath)
		args = []string{"neww", "-c", projPath, "-n", leafName}
		if outer != "" {
			args = append(args, "-t", outer)
		}
		args = append(args, "--", "claude")
	}
	_ = exec.Command("tmux", args...).Run()
	_ = tmuxclient.DetachIfShpell()
	return m, nil
}

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
	listLen := m.filteredListLen()
	if m.cursor >= listLen {
		m.cursor = max(0, listLen-1)
	}
}

func (m *model) clampCursorToFiltered() {
	listLen := m.filteredListLen()
	if m.cursor >= listLen {
		m.cursor = max(0, listLen-1)
	}
}

func (m model) filteredListLen() int {
	if m.activeView == viewNotifications {
		return len(m.filteredEntries())
	}
	if m.drillProject != "" {
		return len(m.filteredDrill())
	}
	return len(buildProjRows(m.filteredSessions()))
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

func (m model) filteredEntries() []entry {
	if m.searchQuery == "" {
		return m.entries
	}
	targets := make([]string, len(m.entries))
	for i, e := range m.entries {
		targets[i] = e.record.WindowName + " " + e.Path + " " + e.record.Session
	}
	matches := fuzzy.Find(m.searchQuery, targets)
	result := make([]entry, 0, len(matches))
	for _, match := range matches {
		result = append(result, m.entries[match.Index])
	}
	return result
}

func (m model) filteredSessions() []sessionEntry {
	if m.searchQuery == "" {
		return m.sessionItems
	}
	targets := make([]string, len(m.sessionItems))
	for i, e := range m.sessionItems {
		targets[i] = e.projPath
	}
	matches := fuzzy.Find(m.searchQuery, targets)
	result := make([]sessionEntry, 0, len(matches))
	for _, match := range matches {
		result = append(result, m.sessionItems[match.Index])
	}
	return result
}

func (m model) filteredDrill() []sessionEntry {
	drilled := m.sessionsForDrill()
	if m.searchQuery == "" {
		return drilled
	}
	targets := make([]string, len(drilled))
	for i, e := range drilled {
		targets[i] = e.record.SessionID + " " + e.record.WindowName
	}
	matches := fuzzy.Find(m.searchQuery, targets)
	result := make([]sessionEntry, 0, len(matches))
	for _, match := range matches {
		result = append(result, drilled[match.Index])
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
	if m.searchMode {
		prefix := lipgloss.NewStyle().Foreground(accent).Render("/ ")
		b.WriteString(prefix + m.searchInput.View() + "\n")
	}
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
	b.WriteString(m.vp.View() + "\n")
	b.WriteString(m.renderFooter())
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
	default:
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

// renderNotificationsContent returns the scrollable content for the
// Notifications view (column header + separator + all data rows).
func (m model) renderNotificationsContent() string {
	var b strings.Builder
	filtered := m.filteredEntries()
	if len(m.entries) == 0 {
		b.WriteString(dimStyle.Render("No pending notifications.") + "\n")
	} else if len(filtered) == 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("No results for %q", m.searchQuery)) + "\n")
	} else {
		pathWidth := max(10, m.termWidth()-74)
		header := fmt.Sprintf("  %-12s  %-3s  %s  %-20s  %-*s  %-14s  %s",
			"STATUS", "PIN", "P", "WINDOW", pathWidth, "PATH", "SESSION", "AGE")
		b.WriteString(dimStyle.Render(header) + "\n")
		b.WriteString(dimStyle.Render("  "+strings.Repeat("─", m.termWidth()-2)) + "\n")
		for i, e := range filtered {
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
				b.WriteString(ind + badge + sep + pin + sep + pop + sep + win + sep + ps + sep + sess + sep + age + pad + "\n")
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
	}
	return b.String()
}

// renderSessionsL1Content returns the scrollable content for the Sessions
// view at Level-1 (projects table).
func (m model) renderSessionsL1Content() string {
	var b strings.Builder
	rows := buildProjRows(m.filteredSessions())
	if len(m.sessionItems) == 0 {
		msg := "No sessions discovered."
		if m.filterActive {
			msg = "No active sessions."
		}
		b.WriteString(dimStyle.Render(msg) + "\n")
		return b.String()
	}
	if len(rows) == 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("No results for %q", m.searchQuery)) + "\n")
		return b.String()
	}
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
			b.WriteString(ind + badge + sep + prj + sep + cnt + sep + age + pad + "\n")
		} else {
			badge := renderStatusBadge(status)
			line := fmt.Sprintf("%s  %s  %5d  %s", badge, proj, row.count, formatAge(row.lastActivity))
			b.WriteString("  " + normalStyle.Render(line) + "\n")
		}
	}
	return b.String()
}

// renderSessionsL2Content returns the scrollable content for the Sessions
// view at Level-2 (session rows for the drilled project).
func (m model) renderSessionsL2Content() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("  ← "+m.drillProject) + "\n")
	b.WriteString(dimStyle.Render("  "+strings.Repeat("─", min(runewidth.StringWidth(m.drillProject)+4, 50))) + "\n")

	drilled := m.filteredDrill()
	if len(m.sessionsForDrill()) == 0 {
		b.WriteString(dimStyle.Render("  No sessions.") + "\n")
	} else if len(drilled) == 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  No results for %q", m.searchQuery)) + "\n")
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
				b.WriteString(ind + badge + sep + pin + sep + sidStr + sep + age + pad + "\n")
			} else {
				badge := renderStatusBadge(status)
				line := fmt.Sprintf("%s  %s  %-10s  %s", badge, pinCol, sid, formatAge(r.LastActivity))
				b.WriteString("  " + normalStyle.Render(line) + "\n")
			}
		}
	}
	return b.String()
}

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
