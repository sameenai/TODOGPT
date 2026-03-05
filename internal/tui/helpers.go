package tui

import (
	"strings"

	"github.com/todogpt/daily-briefing/internal/models"
)

// Section indices.
const (
	secNews = iota
	secWeather
	secCalendar
	secEmail
	secSlack
	secGitHub
	secTodos
	numSections
)

var sectionNames = [numSections]string{
	"News", "Weather", "Calendar", "Email", "Slack", "GitHub", "Todos",
}

// countPending returns the number of pending/in-progress todos.
func countPending(todos []models.TodoItem) int {
	n := 0
	for _, t := range todos {
		if t.Status == models.TodoPending || t.Status == models.TodoInProgress {
			n++
		}
	}
	return n
}

// filterPending returns only pending/in-progress todos.
func filterPending(todos []models.TodoItem) []models.TodoItem {
	var out []models.TodoItem
	for _, t := range todos {
		if t.Status == models.TodoPending || t.Status == models.TodoInProgress {
			out = append(out, t)
		}
	}
	return out
}

// padLines pads s with blank lines until it occupies exactly height lines.
func padLines(s string, height int) string {
	lines := strings.Split(s, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// truncate shortens s to at most max runes, appending "..." when cut.
func truncate(s string, max int) string {
	if max <= 3 || len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
