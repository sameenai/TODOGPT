package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
	"github.com/todogpt/daily-briefing/internal/services"
)

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("236"))

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("14")).
			Background(lipgloss.Color("237")).
			Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Background(lipgloss.Color("234")).
				Padding(0, 1)

	tabBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("234"))

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Background(lipgloss.Color("234"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("14"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	urgentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)

	highStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")).
			Bold(true)
)

// ── Messages ──────────────────────────────────────────────────────────────────

// fetchDoneMsg carries the result of a background FetchAll call.
type fetchDoneMsg struct{ briefing *models.Briefing }

type tickMsg time.Time

// pomodoroTickMsg fires every second when the pomodoro timer is running.
type pomodoroTickMsg time.Time

// ── Model ─────────────────────────────────────────────────────────────────────

// model is the bubbletea application model for the dashboard TUI.
type model struct {
	hub          *services.Hub
	briefing     *models.Briefing
	activeTab    int
	scroll       int
	selectedTodo int
	width        int
	height       int
	loading      bool
	lastFetch    time.Time
	// Todo input mode (n key opens inline new-todo prompt)
	inputMode bool
	inputText string
	// Pomodoro timer
	pomodoroCfg     config.PomodoroConfig
	pomodoroRunning bool
	pomodoroWork    bool // true = work phase, false = break phase
	pomodoroLeft    time.Duration
}

// newModel constructs the initial model for a given Hub and pomodoro config.
func newModel(hub *services.Hub, pomodoro config.PomodoroConfig) model {
	workDur := time.Duration(pomodoro.WorkMinutes) * time.Minute
	if workDur == 0 {
		workDur = 25 * time.Minute
	}
	return model{
		hub:          hub,
		loading:      true,
		pomodoroCfg:  pomodoro,
		pomodoroWork: true,
		pomodoroLeft: workDur,
	}
}

// Init implements tea.Model – kicks off an initial fetch and periodic tick.
func (m model) Init() tea.Cmd {
	return tea.Batch(doFetch(m.hub), scheduleTick())
}

func schedulePomodoroTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return pomodoroTickMsg(t) })
}

func doFetch(hub *services.Hub) tea.Cmd {
	return func() tea.Msg {
		return fetchDoneMsg{briefing: hub.FetchAll()}
	}
}

func scheduleTick() tea.Cmd {
	return tea.Tick(5*time.Minute, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// ── Update ────────────────────────────────────────────────────────────────────

// Update implements tea.Model.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case fetchDoneMsg:
		m.briefing = msg.briefing
		m.loading = false
		m.lastFetch = time.Now()
		m.scroll = 0
		m.selectedTodo = 0

	case tickMsg:
		return m, tea.Batch(doFetch(m.hub), scheduleTick())

	case pomodoroTickMsg:
		if m.pomodoroRunning {
			m.pomodoroLeft -= time.Second
			if m.pomodoroLeft <= 0 {
				// Phase complete — switch phases
				m.pomodoroWork = !m.pomodoroWork
				breakDur := time.Duration(m.pomodoroCfg.BreakMinutes) * time.Minute
				if breakDur == 0 {
					breakDur = 5 * time.Minute
				}
				workDur := time.Duration(m.pomodoroCfg.WorkMinutes) * time.Minute
				if workDur == 0 {
					workDur = 25 * time.Minute
				}
				if m.pomodoroWork {
					m.pomodoroLeft = workDur
				} else {
					m.pomodoroLeft = breakDur
				}
			}
			return m, schedulePomodoroTick()
		}

	case tea.KeyMsg:
		key := msg.String()
		// In input mode, only esc/ctrl+c quit; all other keys go to input handler
		if m.inputMode {
			if key == "ctrl+c" {
				return m, tea.Quit
			}
			return m.applyInputKey(key), nil
		}
		if key == "q" || key == "ctrl+c" {
			return m, tea.Quit
		}
		next := m.applyKey(key)
		// If pomodoro just started, kick off the per-second tick.
		if key == "p" && next.pomodoroRunning && !m.pomodoroRunning {
			return next, schedulePomodoroTick()
		}
		return next, nil
	}
	return m, nil
}

// applyKey is the pure key-dispatch function; it is separated so tests can
// drive it directly without constructing bubbletea messages.
func (m model) applyKey(key string) model {
	switch key {
	case "tab", "right", "l":
		m.activeTab = (m.activeTab + 1) % numSections
		m.scroll = 0
		m.selectedTodo = 0

	case "shift+tab", "left", "h":
		m.activeTab = (m.activeTab - 1 + numSections) % numSections
		m.scroll = 0
		m.selectedTodo = 0

	case "1", "2", "3", "4", "5", "6", "7":
		m.activeTab = int(key[0] - '1')
		m.scroll = 0
		m.selectedTodo = 0

	case "up", "k":
		if m.activeTab == secTodos {
			if m.selectedTodo > 0 {
				m.selectedTodo--
			}
		} else if m.scroll > 0 {
			m.scroll--
		}

	case "down", "j":
		if m.activeTab == secTodos && m.briefing != nil {
			n := countPending(m.briefing.Todos)
			if m.selectedTodo < n-1 {
				m.selectedTodo++
			}
		} else {
			m.scroll++
		}

	case " ", "enter":
		if m.activeTab == secTodos && m.briefing != nil && m.hub != nil {
			pending := filterPending(m.briefing.Todos)
			if m.selectedTodo < len(pending) {
				m.hub.Todos.Complete(pending[m.selectedTodo].ID)
				m.briefing.Todos = m.hub.Todos.List()
				n := countPending(m.briefing.Todos)
				if m.selectedTodo >= n && m.selectedTodo > 0 {
					m.selectedTodo = n - 1
				}
			}
		}

	case "n":
		if m.activeTab == secTodos {
			m.inputMode = true
			m.inputText = ""
		}

	case "d":
		if m.activeTab == secTodos && m.briefing != nil && m.hub != nil {
			pending := filterPending(m.briefing.Todos)
			if m.selectedTodo < len(pending) {
				m.hub.Todos.Delete(pending[m.selectedTodo].ID)
				m.briefing.Todos = m.hub.Todos.List()
				n := countPending(m.briefing.Todos)
				if m.selectedTodo >= n && m.selectedTodo > 0 {
					m.selectedTodo = n - 1
				}
			}
		}

	case "i":
		if m.activeTab == secTodos && m.briefing != nil && m.hub != nil {
			pending := filterPending(m.briefing.Todos)
			if m.selectedTodo < len(pending) {
				todo := pending[m.selectedTodo]
				if todo.Status == models.TodoInProgress {
					m.hub.Todos.Update(todo.ID, func(t *models.TodoItem) {
						t.Status = models.TodoPending
					})
				} else {
					m.hub.Todos.Update(todo.ID, func(t *models.TodoItem) {
						t.Status = models.TodoInProgress
					})
				}
				m.briefing.Todos = m.hub.Todos.List()
			}
		}

	case "r":
		m.loading = true

	case "p":
		if m.pomodoroCfg.Enabled {
			m.pomodoroRunning = !m.pomodoroRunning
		}
	}
	return m
}

// applyInputKey handles keystrokes when the new-todo input prompt is open.
func (m model) applyInputKey(key string) model {
	switch key {
	case "esc":
		m.inputMode = false
		m.inputText = ""
	case "enter":
		if m.inputText != "" && m.hub != nil {
			m.hub.Todos.Add(models.TodoItem{
				Title:    m.inputText,
				Priority: models.PriorityMedium,
				Status:   models.TodoPending,
				Source:   "manual",
			})
			if m.briefing != nil {
				m.briefing.Todos = m.hub.Todos.List()
			}
		}
		m.inputMode = false
		m.inputText = ""
	case "backspace":
		if len(m.inputText) > 0 {
			m.inputText = m.inputText[:len(m.inputText)-1]
		}
	default:
		// Accept printable single-rune keys
		if len(key) == 1 {
			m.inputText += key
		}
	}
	return m
}

// ── View ──────────────────────────────────────────────────────────────────────

// View implements tea.Model.
func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	header := m.viewHeader()
	tabs := m.viewTabs()
	statusBar := m.viewStatusBar()

	headerH := strings.Count(header, "\n") + 1
	tabsH := strings.Count(tabs, "\n") + 1
	statusH := strings.Count(statusBar, "\n") + 1
	contentH := m.height - headerH - tabsH - statusH
	if contentH < 1 {
		contentH = 1
	}

	content := m.viewSection(contentH)
	return strings.Join([]string{header, tabs, content, statusBar}, "\n")
}

func (m model) viewHeader() string {
	now := time.Now()
	greeting := "Good morning"
	if h := now.Hour(); h >= 12 && h < 17 {
		greeting = "Good afternoon"
	} else if h >= 17 {
		greeting = "Good evening"
	}

	left := fmt.Sprintf("  %s! %s", greeting, now.Format("Monday, January 2"))
	var right string
	if m.loading {
		right = "⟳ Refreshing...  "
	} else if !m.lastFetch.IsZero() {
		right = fmt.Sprintf("Updated %s  ", m.lastFetch.Format("3:04:05 PM"))
	}

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	return headerStyle.Width(m.width).Render(left + strings.Repeat(" ", gap) + right)
}

func (m model) viewTabs() string {
	var parts []string
	for i, name := range sectionNames {
		label := name
		if badge := m.badge(i); badge != "" {
			label += " " + badge
		}
		if i == m.activeTab {
			parts = append(parts, activeTabStyle.Render(label))
		} else {
			parts = append(parts, inactiveTabStyle.Render(label))
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	pad := m.width - lipgloss.Width(row)
	if pad > 0 {
		row += tabBarStyle.Render(strings.Repeat(" ", pad))
	}
	return row
}

func (m model) badge(sec int) string {
	if m.briefing == nil {
		return ""
	}
	b := m.briefing
	switch sec {
	case secNews:
		if n := len(b.News); n > 0 {
			return dimStyle.Render(fmt.Sprintf("(%d)", n))
		}
	case secCalendar:
		if n := len(b.Events); n > 0 {
			return dimStyle.Render(fmt.Sprintf("(%d)", n))
		}
	case secEmail:
		n := 0
		for _, e := range b.UnreadEmails {
			if e.IsUnread {
				n++
			}
		}
		if n > 0 {
			return urgentStyle.Render(fmt.Sprintf("(%d)", n))
		}
	case secSlack:
		if n := len(b.SlackMessages); n > 0 {
			return dimStyle.Render(fmt.Sprintf("(%d)", n))
		}
	case secGitHub:
		n := 0
		for _, notif := range b.GitHubNotifs {
			if notif.Unread {
				n++
			}
		}
		if n > 0 {
			return dimStyle.Render(fmt.Sprintf("(%d)", n))
		}
	case secTodos:
		if n := countPending(b.Todos); n > 0 {
			return urgentStyle.Render(fmt.Sprintf("(%d)", n))
		}
	}
	return ""
}

func (m model) viewStatusBar() string {
	var keys []string
	if m.inputMode {
		keys = append(keys, "type title", "enter confirm", "esc cancel")
	} else if m.activeTab == secTodos {
		keys = append(keys, "↑/↓ select", "space/enter done", "i in-progress", "d delete", "n new")
	} else {
		keys = append(keys, "↑/↓ scroll")
	}
	if !m.inputMode {
		keys = append(keys, "←/→ tab", "1-7 jump", "r refresh", "q quit")
	}
	left := "  " + strings.Join(keys, "  ·  ")

	var right string
	if m.pomodoroCfg.Enabled {
		mins := int(m.pomodoroLeft.Minutes())
		secs := int(m.pomodoroLeft.Seconds()) % 60
		phase := "work"
		icon := "🍅"
		if !m.pomodoroWork {
			phase = "break"
			icon = "☕"
		}
		state := fmt.Sprintf("%s %02d:%02d %s", icon, mins, secs, phase)
		if !m.pomodoroRunning {
			state += " (p to start)"
		}
		right = "  " + state + "  "
	}

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	return statusBarStyle.Width(m.width).Render(left + strings.Repeat(" ", gap) + right)
}

func (m model) viewSection(height int) string {
	if m.loading && m.briefing == nil {
		return padLines("  Fetching data...", height)
	}
	if m.briefing == nil {
		return padLines("  No data. Press r to refresh.", height)
	}

	var raw string
	switch m.activeTab {
	case secNews:
		raw = m.renderNews()
	case secWeather:
		raw = m.renderWeather()
	case secCalendar:
		raw = m.renderCalendar()
	case secEmail:
		raw = m.renderEmail()
	case secSlack:
		raw = m.renderSlack()
	case secGitHub:
		raw = m.renderGitHub()
	case secTodos:
		raw = m.renderTodos()
	}

	lines := strings.Split(raw, "\n")
	start := m.scroll
	if start > len(lines) {
		start = len(lines)
	}
	lines = lines[start:]
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// Run starts the bubbletea TUI. Called from cmd/tui/main.go.
func Run(hub *services.Hub, pomodoro config.PomodoroConfig) error {
	p := tea.NewProgram(newModel(hub, pomodoro), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
