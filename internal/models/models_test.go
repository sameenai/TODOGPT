package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPriorityString(t *testing.T) {
	tests := []struct {
		priority Priority
		expected string
	}{
		{PriorityLow, "low"},
		{PriorityMedium, "medium"},
		{PriorityHigh, "high"},
		{PriorityUrgent, "urgent"},
		{Priority(99), "medium"}, // unknown defaults to medium
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.priority.String()
			if got != tt.expected {
				t.Errorf("Priority(%d).String() = %q, want %q", tt.priority, got, tt.expected)
			}
		})
	}
}

func TestTodoStatusString(t *testing.T) {
	tests := []struct {
		status   TodoStatus
		expected string
	}{
		{TodoPending, "pending"},
		{TodoInProgress, "in_progress"},
		{TodoDone, "done"},
		{TodoArchived, "archived"},
		{TodoStatus(99), "pending"}, // unknown defaults to pending
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.status.String()
			if got != tt.expected {
				t.Errorf("TodoStatus(%d).String() = %q, want %q", tt.status, got, tt.expected)
			}
		})
	}
}

func TestPriorityConstants(t *testing.T) {
	if PriorityLow != 0 {
		t.Errorf("PriorityLow should be 0, got %d", PriorityLow)
	}
	if PriorityMedium != 1 {
		t.Errorf("PriorityMedium should be 1, got %d", PriorityMedium)
	}
	if PriorityHigh != 2 {
		t.Errorf("PriorityHigh should be 2, got %d", PriorityHigh)
	}
	if PriorityUrgent != 3 {
		t.Errorf("PriorityUrgent should be 3, got %d", PriorityUrgent)
	}
}

func TestTodoStatusConstants(t *testing.T) {
	if TodoPending != 0 {
		t.Errorf("TodoPending should be 0, got %d", TodoPending)
	}
	if TodoInProgress != 1 {
		t.Errorf("TodoInProgress should be 1, got %d", TodoInProgress)
	}
	if TodoDone != 2 {
		t.Errorf("TodoDone should be 2, got %d", TodoDone)
	}
	if TodoArchived != 3 {
		t.Errorf("TodoArchived should be 3, got %d", TodoArchived)
	}
}

func TestWeatherJSON(t *testing.T) {
	w := Weather{
		City:        "Tokyo",
		Temperature: 72.5,
		FeelsLike:   70.0,
		Humidity:    55,
		Description: "clear sky",
		Icon:        "01d",
		WindSpeed:   5.5,
		Units:       "imperial",
		UpdatedAt:   time.Now(),
	}

	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Weather
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.City != "Tokyo" {
		t.Errorf("expected city Tokyo, got %s", decoded.City)
	}
	if decoded.Temperature != 72.5 {
		t.Errorf("expected temp 72.5, got %f", decoded.Temperature)
	}
	if decoded.Humidity != 55 {
		t.Errorf("expected humidity 55, got %d", decoded.Humidity)
	}
}

func TestCalendarEventJSON(t *testing.T) {
	start := time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	evt := CalendarEvent{
		ID:         "evt-1",
		Title:      "Team Standup",
		StartTime:  start,
		EndTime:    end,
		AllDay:     false,
		MeetingURL: "https://zoom.us/j/test",
		Attendees:  []string{"alice@co.com", "bob@co.com"},
		Source:     "google_calendar",
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded CalendarEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.ID != "evt-1" {
		t.Errorf("expected ID evt-1, got %s", decoded.ID)
	}
	if decoded.Title != "Team Standup" {
		t.Errorf("expected title Team Standup, got %s", decoded.Title)
	}
	if len(decoded.Attendees) != 2 {
		t.Errorf("expected 2 attendees, got %d", len(decoded.Attendees))
	}
	if decoded.AllDay {
		t.Error("expected AllDay false")
	}
}

func TestTodoItemJSON(t *testing.T) {
	now := time.Now()
	due := now.Add(24 * time.Hour)
	item := TodoItem{
		ID:          "todo-1",
		Title:       "Review PR",
		Description: "Check the auth changes",
		Priority:    PriorityHigh,
		Status:      TodoPending,
		Source:      "github",
		SourceID:    "gh-123",
		DueDate:     &due,
		CreatedAt:   now,
		UpdatedAt:   now,
		Tags:        []string{"code-review", "urgent"},
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded TodoItem
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Priority != PriorityHigh {
		t.Errorf("expected priority High, got %d", decoded.Priority)
	}
	if decoded.Status != TodoPending {
		t.Errorf("expected status Pending, got %d", decoded.Status)
	}
	if decoded.DueDate == nil {
		t.Error("expected due date to be set")
	}
	if len(decoded.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(decoded.Tags))
	}
}

func TestTodoItemNilDueDate(t *testing.T) {
	item := TodoItem{
		ID:     "todo-2",
		Title:  "No due date",
		Status: TodoPending,
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Verify due_date is omitted
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)
	if _, exists := raw["due_date"]; exists {
		t.Error("expected due_date to be omitted when nil")
	}
}

func TestBriefingJSON(t *testing.T) {
	b := Briefing{
		Date:        time.Now(),
		EmailCount:  5,
		SlackUnread: 3,
		GeneratedAt: time.Now(),
		Events:      []CalendarEvent{},
		News:        []NewsItem{},
	}

	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Briefing
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.EmailCount != 5 {
		t.Errorf("expected email count 5, got %d", decoded.EmailCount)
	}
	if decoded.SlackUnread != 3 {
		t.Errorf("expected slack unread 3, got %d", decoded.SlackUnread)
	}
}

func TestDashboardUpdateJSON(t *testing.T) {
	update := DashboardUpdate{
		Type:    "full_refresh",
		Payload: map[string]string{"key": "value"},
	}

	data, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded DashboardUpdate
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Type != "full_refresh" {
		t.Errorf("expected type full_refresh, got %s", decoded.Type)
	}
}

func TestSignalJSON(t *testing.T) {
	sig := Signal{
		ID:       "sig-1",
		Source:   "slack",
		Type:     "message",
		Title:    "New message",
		Body:     "Hello there",
		Priority: PriorityHigh,
		Read:     false,
	}

	data, err := json.Marshal(sig)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Signal
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Source != "slack" {
		t.Errorf("expected source slack, got %s", decoded.Source)
	}
	if decoded.Priority != PriorityHigh {
		t.Errorf("expected priority High, got %d", decoded.Priority)
	}
	if decoded.Read {
		t.Error("expected read false")
	}
}

func TestEmailMessageJSON(t *testing.T) {
	email := EmailMessage{
		ID:       "em-1",
		From:     "test@example.com",
		Subject:  "Important Update",
		IsUnread: true,
		Labels:   []string{"inbox", "important"},
	}

	data, err := json.Marshal(email)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded EmailMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if !decoded.IsUnread {
		t.Error("expected IsUnread true")
	}
	if len(decoded.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(decoded.Labels))
	}
}

func TestSlackMessageJSON(t *testing.T) {
	msg := SlackMessage{
		Channel:  "#engineering",
		User:     "alice",
		Text:     "Deploy complete",
		IsUrgent: true,
		IsDM:     false,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded SlackMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Channel != "#engineering" {
		t.Errorf("expected channel #engineering, got %s", decoded.Channel)
	}
	if !decoded.IsUrgent {
		t.Error("expected IsUrgent true")
	}
}

func TestGitHubNotificationJSON(t *testing.T) {
	notif := GitHubNotification{
		ID:     "gh-1",
		Title:  "Fix bug",
		Repo:   "org/repo",
		Type:   "PullRequest",
		Reason: "review_requested",
		Unread: true,
	}

	data, err := json.Marshal(notif)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded GitHubNotification
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Type != "PullRequest" {
		t.Errorf("expected type PullRequest, got %s", decoded.Type)
	}
	if !decoded.Unread {
		t.Error("expected Unread true")
	}
}
