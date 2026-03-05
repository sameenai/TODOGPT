package tui

import (
	"sort"
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

// Sort mode constants for the Todos tab.
const (
	sortByPriority = iota // default: highest priority first
	sortByStatus          // in-progress first, then pending
	sortByTitle           // alphabetical
	numSortModes
)

var sortModeNames = [numSortModes]string{"priority", "status", "title"}

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

// sortAndFilterTodos applies keyword filtering then sorts pending todos by mode.
func sortAndFilterTodos(todos []models.TodoItem, filter string, mode int) []models.TodoItem {
	pending := filterPending(todos)

	if filter != "" {
		lower := strings.ToLower(filter)
		var matched []models.TodoItem
		for _, t := range pending {
			if strings.Contains(strings.ToLower(t.Title), lower) ||
				strings.Contains(strings.ToLower(t.Source), lower) {
				matched = append(matched, t)
			}
		}
		pending = matched
	}

	switch mode {
	case sortByPriority:
		sort.SliceStable(pending, func(i, j int) bool {
			return pending[i].Priority > pending[j].Priority
		})
	case sortByStatus:
		sort.SliceStable(pending, func(i, j int) bool {
			// in-progress (2) before pending (1)
			return pending[i].Status > pending[j].Status
		})
	case sortByTitle:
		sort.SliceStable(pending, func(i, j int) bool {
			return strings.ToLower(pending[i].Title) < strings.ToLower(pending[j].Title)
		})
	}
	return pending
}
