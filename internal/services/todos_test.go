package services

import (
	"testing"
	"time"

	"github.com/todogpt/daily-briefing/internal/models"
)

func TestNewTodoService(t *testing.T) {
	svc := NewTodoService()
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	items := svc.List()
	if len(items) != 0 {
		t.Errorf("expected empty list, got %d items", len(items))
	}
}

func TestTodoAdd(t *testing.T) {
	svc := NewTodoService()
	svc.Add(models.TodoItem{
		Title:    "Test Task",
		Priority: models.PriorityHigh,
	})

	items := svc.List()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Title != "Test Task" {
		t.Errorf("expected title 'Test Task', got %q", items[0].Title)
	}
	if items[0].ID == "" {
		t.Error("expected auto-generated ID")
	}
	if items[0].CreatedAt.IsZero() {
		t.Error("expected auto-set CreatedAt")
	}
	if items[0].UpdatedAt.IsZero() {
		t.Error("expected auto-set UpdatedAt")
	}
}

func TestTodoAddWithExistingID(t *testing.T) {
	svc := NewTodoService()
	svc.Add(models.TodoItem{
		ID:    "custom-id",
		Title: "Custom ID Task",
	})

	items := svc.List()
	if items[0].ID != "custom-id" {
		t.Errorf("expected ID 'custom-id', got %q", items[0].ID)
	}
}

func TestTodoUpdate(t *testing.T) {
	svc := NewTodoService()
	svc.Add(models.TodoItem{ID: "upd-1", Title: "Before"})

	ok := svc.Update("upd-1", func(item *models.TodoItem) {
		item.Title = "After"
		item.Priority = models.PriorityUrgent
	})
	if !ok {
		t.Error("expected update to return true")
	}

	items := svc.List()
	if items[0].Title != "After" {
		t.Errorf("expected title 'After', got %q", items[0].Title)
	}
	if items[0].Priority != models.PriorityUrgent {
		t.Errorf("expected priority Urgent, got %d", items[0].Priority)
	}
}

func TestTodoUpdateNonExistent(t *testing.T) {
	svc := NewTodoService()
	ok := svc.Update("ghost", func(item *models.TodoItem) {
		item.Title = "Should not happen"
	})
	if ok {
		t.Error("expected update to return false for non-existent item")
	}
}

func TestTodoDelete(t *testing.T) {
	svc := NewTodoService()
	svc.Add(models.TodoItem{ID: "del-1", Title: "Delete Me"})
	svc.Add(models.TodoItem{ID: "del-2", Title: "Keep Me"})

	ok := svc.Delete("del-1")
	if !ok {
		t.Error("expected delete to return true")
	}

	items := svc.List()
	if len(items) != 1 {
		t.Errorf("expected 1 item after delete, got %d", len(items))
	}
	if items[0].ID != "del-2" {
		t.Errorf("wrong item remaining: %s", items[0].ID)
	}
}

func TestTodoDeleteNonExistent(t *testing.T) {
	svc := NewTodoService()
	ok := svc.Delete("ghost")
	if ok {
		t.Error("expected delete to return false for non-existent item")
	}
}

func TestTodoComplete(t *testing.T) {
	svc := NewTodoService()
	svc.Add(models.TodoItem{ID: "comp-1", Title: "Complete Me", Status: models.TodoPending})

	ok := svc.Complete("comp-1")
	if !ok {
		t.Error("expected complete to return true")
	}

	items := svc.List()
	if items[0].Status != models.TodoDone {
		t.Errorf("expected status Done, got %d", items[0].Status)
	}
	if items[0].CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

func TestTodoSetItems(t *testing.T) {
	svc := NewTodoService()
	svc.Add(models.TodoItem{ID: "old", Title: "Old"})

	newItems := []models.TodoItem{
		{ID: "new-1", Title: "New 1"},
		{ID: "new-2", Title: "New 2"},
	}
	svc.SetItems(newItems)

	items := svc.List()
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestTodoListReturnsCopy(t *testing.T) {
	svc := NewTodoService()
	svc.Add(models.TodoItem{ID: "orig", Title: "Original"})

	items := svc.List()
	items[0].Title = "Modified"

	items2 := svc.List()
	if items2[0].Title != "Original" {
		t.Error("List() should return a copy, not a reference")
	}
}

func TestGenerateFromBriefing(t *testing.T) {
	svc := NewTodoService()
	now := time.Now()

	briefing := &models.Briefing{
		UnreadEmails: []models.EmailMessage{
			{
				ID:       "e1",
				From:     "boss@co.com",
				Subject:  "Action Required: Review Q1",
				IsUnread: true,
				Labels:   []string{"important"},
				Date:     now,
			},
			{
				ID:       "e2",
				From:     "team@co.com",
				Subject:  "Meeting notes",
				IsUnread: true,
				Labels:   []string{"inbox"},
				Date:     now,
			},
		},
		SlackMessages: []models.SlackMessage{
			{
				Channel:   "DM",
				User:      "alice",
				Text:      "Can you review my PR?",
				Timestamp: now,
				IsDM:      true,
			},
			{
				Channel:   "#incidents",
				User:      "bot",
				Text:      "CRITICAL: service down",
				Timestamp: now,
				IsUrgent:  true,
			},
			{
				Channel:   "#random",
				User:      "bob",
				Text:      "Anyone for lunch?",
				Timestamp: now,
			},
		},
		GitHubNotifs: []models.GitHubNotification{
			{
				ID:        "gh1",
				Title:     "Fix memory leak",
				Repo:      "org/repo",
				Type:      "PullRequest",
				Reason:    "review_requested",
				Unread:    true,
				UpdatedAt: now,
			},
			{
				ID:        "gh2",
				Title:     "Old issue",
				Repo:      "org/repo",
				Type:      "Issue",
				Reason:    "mention",
				Unread:    false, // should be skipped
				UpdatedAt: now,
			},
		},
		Events: []models.CalendarEvent{
			{
				ID:          "evt1",
				Title:       "Sprint Planning",
				Description: "Prepare sprint backlog",
				StartTime:   now.Add(2 * time.Hour),
			},
			{
				ID:    "evt2",
				Title: "Standup",
				// No description — should be skipped
				StartTime: now.Add(1 * time.Hour),
			},
		},
	}

	svc.GenerateFromBriefing(briefing)
	items := svc.List()

	// Expected: 2 emails + 2 slack (DM + urgent, not #random) + 1 github (unread only) + 1 calendar (with description)
	expectedCount := 6
	if len(items) != expectedCount {
		t.Errorf("expected %d auto-generated todos, got %d", expectedCount, len(items))
		for _, item := range items {
			t.Logf("  - [%s] %s (source: %s)", item.Priority.String(), item.Title, item.Source)
		}
	}
}

func TestGenerateFromBriefingDeduplication(t *testing.T) {
	svc := NewTodoService()
	now := time.Now()

	briefing := &models.Briefing{
		UnreadEmails: []models.EmailMessage{
			{ID: "e1", From: "a@b.com", Subject: "Test", IsUnread: true, Labels: []string{"inbox"}, Date: now},
		},
	}

	svc.GenerateFromBriefing(briefing)
	count1 := len(svc.List())

	// Call again — should not duplicate
	svc.GenerateFromBriefing(briefing)
	count2 := len(svc.List())

	if count2 != count1 {
		t.Errorf("expected deduplication: first=%d, second=%d", count1, count2)
	}
}

func TestGenerateFromBriefingPriorityDetection(t *testing.T) {
	svc := NewTodoService()
	now := time.Now()

	briefing := &models.Briefing{
		UnreadEmails: []models.EmailMessage{
			{
				ID:       "urgent-email",
				From:     "boss@co.com",
				Subject:  "URGENT: Server down",
				IsUnread: true,
				Labels:   []string{"inbox"},
				Date:     now,
			},
			{
				ID:        "starred-email",
				From:      "lead@co.com",
				Subject:   "Architecture decision",
				IsUnread:  true,
				IsStarred: true,
				Labels:    []string{"inbox"},
				Date:      now,
			},
		},
	}

	svc.GenerateFromBriefing(briefing)
	items := svc.List()

	for _, item := range items {
		if item.SourceID == "urgent-email" && item.Priority != models.PriorityUrgent {
			t.Errorf("urgent email should have Urgent priority, got %s", item.Priority.String())
		}
		if item.SourceID == "starred-email" && item.Priority != models.PriorityHigh {
			t.Errorf("starred email should have High priority, got %s", item.Priority.String())
		}
	}
}

func TestGenerateFromBriefingEmpty(t *testing.T) {
	svc := NewTodoService()
	briefing := &models.Briefing{}

	svc.GenerateFromBriefing(briefing)
	items := svc.List()

	if len(items) != 0 {
		t.Errorf("expected 0 items from empty briefing, got %d", len(items))
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a longer string", 10, "this is..."},
		{"", 5, ""},
		{"ab", 5, "ab"},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
		}
	}
}
