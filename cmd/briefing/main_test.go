package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/todogpt/daily-briefing/internal/models"
)

func captureOutput(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outCh := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outCh <- buf.String()
	}()

	fn()
	w.Close()
	os.Stdout = old
	return <-outCh
}

func testBriefing() *models.Briefing {
	now := time.Now()
	dueDate := now.Add(24 * time.Hour)
	return &models.Briefing{
		Weather: &models.Weather{
			City:        "Test City",
			Temperature: 72,
			FeelsLike:   70,
			Humidity:    45,
			Description: "sunny",
			WindSpeed:   10,
		},
		Events: []models.CalendarEvent{
			{Title: "Team Standup", StartTime: now, EndTime: now.Add(30 * time.Minute), Location: "Room 101"},
			{Title: "Virtual Call", StartTime: now.Add(2 * time.Hour), EndTime: now.Add(3 * time.Hour), MeetingURL: "https://zoom.us/j/123"},
			{Title: "Lunch Break", StartTime: now.Add(4 * time.Hour), EndTime: now.Add(5 * time.Hour)},
		},
		News: []models.NewsItem{
			{Title: "Go 1.23 Released", Description: "Major performance improvements", URL: "#", Source: "Hacker News", PublishedAt: now},
			{Title: "AI Advances Continue", Description: "New breakthroughs", URL: "#", Source: "TechCrunch", PublishedAt: now},
			{Title: "Cloud Computing Trends", Description: "Cost optimization", URL: "#", Source: "The Verge", PublishedAt: now},
			{Title: "Open Source Update", Description: "Security audit", URL: "#", Source: "Hacker News", PublishedAt: now},
			{Title: "Developer Tools Survey", Description: "Latest results", URL: "#", Source: "Hacker News", PublishedAt: now},
			{Title: "Sixth Article Beyond Limit", Description: "Should not show", URL: "#", Source: "Hacker News", PublishedAt: now},
		},
		UnreadEmails: []models.EmailMessage{
			{Subject: "Urgent: Deploy review needed", From: "alice@example.com", IsUnread: true, IsStarred: true, Date: now},
			{Subject: "Weekly newsletter", From: "news@example.com", IsUnread: true, IsStarred: false, Date: now},
			{Subject: "Already read email", From: "bob@example.com", IsUnread: false, IsStarred: false, Date: now},
		},
		SlackMessages: []models.SlackMessage{
			{Channel: "#general", User: "alice", Text: "Hello team!", Timestamp: now, IsUrgent: true, IsDM: false},
			{Channel: "DM", User: "bob", Text: "Quick question", Timestamp: now, IsUrgent: false, IsDM: true},
			{Channel: "#dev", User: "charlie", Text: "PR merged", Timestamp: now, IsUrgent: false, IsDM: false},
		},
		GitHubNotifs: []models.GitHubNotification{
			{Title: "Fix critical bug in auth module", Repo: "org/backend", Type: "PullRequest", URL: "#", Reason: "review_requested", Unread: true, UpdatedAt: now},
			{Title: "Add unit tests for API layer", Repo: "org/backend", Type: "Issue", URL: "#", Reason: "assigned", Unread: true, UpdatedAt: now},
			{Title: "Read notification", Repo: "org/frontend", Type: "PullRequest", URL: "#", Reason: "mention", Unread: false, UpdatedAt: now},
		},
		Todos: []models.TodoItem{
			{ID: "1", Title: "Deploy hotfix", Priority: models.PriorityUrgent, Status: models.TodoPending, Source: "github", DueDate: &dueDate},
			{ID: "2", Title: "Review PR #42", Priority: models.PriorityHigh, Status: models.TodoInProgress, Source: "github"},
			{ID: "3", Title: "Update docs", Priority: models.PriorityMedium, Status: models.TodoPending, Source: "manual"},
			{ID: "4", Title: "Completed task", Priority: models.PriorityLow, Status: models.TodoDone, Source: "manual"},
			{ID: "5", Title: "Archived task", Priority: models.PriorityLow, Status: models.TodoArchived, Source: "manual"},
		},
		GeneratedAt: now,
		Date:        now,
		EmailCount:  2,
		SlackUnread: 2,
	}
}

func TestPrintHeaderMorning(t *testing.T) {
	morning := time.Date(2024, 6, 15, 9, 0, 0, 0, time.Local)
	output := captureOutput(func() {
		printHeaderAt(morning)
	})
	if !strings.Contains(output, "Good morning") {
		t.Error("expected 'Good morning' greeting for 9 AM")
	}
	if !strings.Contains(output, "Saturday") {
		t.Error("expected day of week in header")
	}
}

func TestPrintHeaderAfternoon(t *testing.T) {
	afternoon := time.Date(2024, 6, 15, 14, 0, 0, 0, time.Local)
	output := captureOutput(func() {
		printHeaderAt(afternoon)
	})
	if !strings.Contains(output, "Good afternoon") {
		t.Error("expected 'Good afternoon' greeting for 2 PM")
	}
}

func TestPrintHeaderEvening(t *testing.T) {
	evening := time.Date(2024, 6, 15, 20, 0, 0, 0, time.Local)
	output := captureOutput(func() {
		printHeaderAt(evening)
	})
	if !strings.Contains(output, "Good evening") {
		t.Error("expected 'Good evening' greeting for 8 PM")
	}
}

func TestPrintHeaderBoundaryNoon(t *testing.T) {
	noon := time.Date(2024, 6, 15, 12, 0, 0, 0, time.Local)
	output := captureOutput(func() {
		printHeaderAt(noon)
	})
	if !strings.Contains(output, "Good afternoon") {
		t.Error("expected 'Good afternoon' at exactly noon")
	}
}

func TestPrintHeaderBoundary5PM(t *testing.T) {
	fivePM := time.Date(2024, 6, 15, 17, 0, 0, 0, time.Local)
	output := captureOutput(func() {
		printHeaderAt(fivePM)
	})
	if !strings.Contains(output, "Good evening") {
		t.Error("expected 'Good evening' at exactly 5 PM")
	}
}

func TestPrintHeaderContainsDate(t *testing.T) {
	t1 := time.Date(2024, 6, 15, 10, 0, 0, 0, time.Local)
	output := captureOutput(func() {
		printHeaderAt(t1)
	})
	if !strings.Contains(output, "June 15, 2024") {
		t.Error("expected formatted date in header")
	}
	if !strings.Contains(output, "=") {
		t.Error("expected separator in header")
	}
}

func TestPrintWeather(t *testing.T) {
	b := testBriefing()
	output := captureOutput(func() {
		printWeather(b)
	})
	if !strings.Contains(output, "Test City") {
		t.Error("expected city in weather output")
	}
	if !strings.Contains(output, "WEATHER") {
		t.Error("expected WEATHER label")
	}
}

func TestPrintWeatherNil(t *testing.T) {
	b := &models.Briefing{}
	output := captureOutput(func() {
		printWeather(b)
	})
	if output != "" {
		t.Error("expected no output when weather is nil")
	}
}

func TestPrintCalendar(t *testing.T) {
	b := testBriefing()
	output := captureOutput(func() {
		printCalendar(b)
	})
	if !strings.Contains(output, "CALENDAR") {
		t.Error("expected CALENDAR label")
	}
	if !strings.Contains(output, "Team Standup") {
		t.Error("expected event title")
	}
	if !strings.Contains(output, "Room 101") {
		t.Error("expected location")
	}
	if !strings.Contains(output, "(virtual)") {
		t.Error("expected virtual indicator for meeting URL event")
	}
}

func TestPrintCalendarEmpty(t *testing.T) {
	b := &models.Briefing{}
	output := captureOutput(func() {
		printCalendar(b)
	})
	if output != "" {
		t.Error("expected no output when events empty")
	}
}

func TestPrintNews(t *testing.T) {
	b := testBriefing()
	output := captureOutput(func() {
		printNews(b)
	})
	if !strings.Contains(output, "TOP NEWS") {
		t.Error("expected TOP NEWS label")
	}
	if !strings.Contains(output, "Go 1.23 Released") {
		t.Error("expected first news title")
	}
	// Should not show 6th item (limit is 5)
	if strings.Contains(output, "Sixth Article Beyond Limit") {
		t.Error("should not show more than 5 news items")
	}
}

func TestPrintNewsEmpty(t *testing.T) {
	b := &models.Briefing{}
	output := captureOutput(func() {
		printNews(b)
	})
	if output != "" {
		t.Error("expected no output when news empty")
	}
}

func TestPrintEmails(t *testing.T) {
	b := testBriefing()
	output := captureOutput(func() {
		printEmails(b)
	})
	if !strings.Contains(output, "EMAIL") {
		t.Error("expected EMAIL label")
	}
	if !strings.Contains(output, "Urgent: Deploy review needed") {
		t.Error("expected unread email subject")
	}
	if !strings.Contains(output, "*") {
		t.Error("expected star marker for starred email")
	}
}

func TestPrintEmailsEmpty(t *testing.T) {
	b := &models.Briefing{}
	output := captureOutput(func() {
		printEmails(b)
	})
	if output != "" {
		t.Error("expected no output when emails empty")
	}
}

func TestPrintSlack(t *testing.T) {
	b := testBriefing()
	output := captureOutput(func() {
		printSlack(b)
	})
	if !strings.Contains(output, "SLACK") {
		t.Error("expected SLACK label")
	}
	if !strings.Contains(output, "#general") {
		t.Error("expected channel name")
	}
	if !strings.Contains(output, "!") {
		t.Error("expected urgent marker")
	}
	if !strings.Contains(output, "@") {
		t.Error("expected DM marker")
	}
}

func TestPrintSlackEmpty(t *testing.T) {
	b := &models.Briefing{}
	output := captureOutput(func() {
		printSlack(b)
	})
	if output != "" {
		t.Error("expected no output when slack empty")
	}
}

func TestPrintGitHub(t *testing.T) {
	b := testBriefing()
	output := captureOutput(func() {
		printGitHub(b)
	})
	if !strings.Contains(output, "GITHUB") {
		t.Error("expected GITHUB label")
	}
	if !strings.Contains(output, "PR") {
		t.Error("expected PR type icon")
	}
	if !strings.Contains(output, "IS") {
		t.Error("expected IS type icon for Issue")
	}
	if !strings.Contains(output, "org/backend") {
		t.Error("expected repo name")
	}
}

func TestPrintGitHubNoUnread(t *testing.T) {
	b := &models.Briefing{
		GitHubNotifs: []models.GitHubNotification{
			{Title: "Read", Unread: false},
		},
	}
	output := captureOutput(func() {
		printGitHub(b)
	})
	if output != "" {
		t.Error("expected no output when no unread notifications")
	}
}

func TestPrintTodos(t *testing.T) {
	b := testBriefing()
	output := captureOutput(func() {
		printTodos(b)
	})
	if !strings.Contains(output, "ACTION ITEMS") {
		t.Error("expected ACTION ITEMS label")
	}
	if !strings.Contains(output, "Deploy hotfix") {
		t.Error("expected urgent todo")
	}
	if !strings.Contains(output, "Review PR #42") {
		t.Error("expected high priority todo")
	}
}

func TestPrintTodosNoPending(t *testing.T) {
	b := &models.Briefing{
		Todos: []models.TodoItem{
			{Status: models.TodoDone, Title: "Done"},
			{Status: models.TodoArchived, Title: "Archived"},
		},
	}
	output := captureOutput(func() {
		printTodos(b)
	})
	if output != "" {
		t.Error("expected no output when no pending todos")
	}
}

func TestPrintFooter(t *testing.T) {
	b := testBriefing()
	output := captureOutput(func() {
		printFooter(b)
	})
	if !strings.Contains(output, "Dashboard") {
		t.Error("expected Dashboard URL in footer")
	}
	if !strings.Contains(output, "Generated at") {
		t.Error("expected generation time in footer")
	}
}

func TestPrintAll(t *testing.T) {
	b := testBriefing()
	output := captureOutput(func() {
		printAll(b)
	})
	if !strings.Contains(output, "Good") {
		t.Error("expected greeting in printAll output")
	}
	if !strings.Contains(output, "WEATHER") {
		t.Error("expected WEATHER section in printAll output")
	}
	if !strings.Contains(output, "CALENDAR") {
		t.Error("expected CALENDAR section in printAll output")
	}
	if !strings.Contains(output, "TOP NEWS") {
		t.Error("expected TOP NEWS section in printAll output")
	}
	if !strings.Contains(output, "EMAIL") {
		t.Error("expected EMAIL section in printAll output")
	}
	if !strings.Contains(output, "SLACK") {
		t.Error("expected SLACK section in printAll output")
	}
	if !strings.Contains(output, "GITHUB") {
		t.Error("expected GITHUB section in printAll output")
	}
	if !strings.Contains(output, "ACTION ITEMS") {
		t.Error("expected ACTION ITEMS section in printAll output")
	}
	if !strings.Contains(output, "Dashboard") {
		t.Error("expected footer in printAll output")
	}
}

func TestRun(t *testing.T) {
	// Use a temp home so run() doesn't depend on real home dir config
	t.Setenv("HOME", t.TempDir())

	output := captureOutput(func() {
		err := run()
		if err != nil {
			t.Errorf("run() returned error: %v", err)
		}
	})
	if !strings.Contains(output, "Good") {
		t.Error("expected greeting in run output")
	}
	if !strings.Contains(output, "Dashboard") {
		t.Error("expected footer in run output")
	}
}

func TestRunConfigError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	dir := filepath.Join(tmpDir, ".daily-briefing")
	path := filepath.Join(dir, "config.json")

	// Write invalid JSON to config in the temp home
	os.MkdirAll(dir, 0755)
	if err := os.WriteFile(path, []byte("invalid{{{"), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	runErr := run()
	if runErr == nil {
		t.Error("expected error from invalid config file")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is..."},
		{"", 5, ""},
		{"abc", 3, "abc"},
		{"abcdef", 6, "abcdef"},
		{"hello world test", 8, "hello..."},
	}
	for _, tc := range tests {
		got := truncate(tc.input, tc.maxLen)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.maxLen, got, tc.want)
		}
	}
}
