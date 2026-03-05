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

// ── Jira todo generation ──────────────────────────────────────────────────────

func TestGenerateFromBriefingJira(t *testing.T) {
	svc := NewTodoService()
	due := time.Now().AddDate(0, 0, 3)
	b := &models.Briefing{
		JiraTickets: []models.JiraTicket{
			{Key: "PROJ-1", Summary: "Fix the login", Status: "To Do", Priority: "Critical", Type: "Bug", DueDate: due},
			{Key: "PROJ-2", Summary: "Add dark mode", Status: "In Progress", Priority: "High", Type: "Story"},
			{Key: "PROJ-3", Summary: "Write docs", Status: "To Do", Priority: "Low", Type: "Task"},
			{Key: "PROJ-4", Summary: "Medium ticket", Status: "To Do", Priority: "Medium", Type: "Task"},
		},
	}
	svc.GenerateFromBriefing(b)
	items := svc.List()
	if len(items) != 4 {
		t.Fatalf("expected 4 jira todos, got %d", len(items))
	}

	// Check priority mapping
	byKey := make(map[string]models.TodoItem)
	for _, item := range items {
		byKey[item.SourceID] = item
	}

	if byKey["PROJ-1"].Priority != models.PriorityUrgent {
		t.Errorf("Critical should map to PriorityUrgent, got %v", byKey["PROJ-1"].Priority)
	}
	if byKey["PROJ-2"].Priority != models.PriorityHigh {
		t.Errorf("High should map to PriorityHigh, got %v", byKey["PROJ-2"].Priority)
	}
	if byKey["PROJ-3"].Priority != models.PriorityLow {
		t.Errorf("Low should map to PriorityLow, got %v", byKey["PROJ-3"].Priority)
	}
	if byKey["PROJ-4"].Priority != models.PriorityMedium {
		t.Errorf("Medium should map to PriorityMedium, got %v", byKey["PROJ-4"].Priority)
	}

	// Check source fields
	t1 := byKey["PROJ-1"]
	if t1.Source != "jira" {
		t.Errorf("expected source=jira, got %s", t1.Source)
	}
	if t1.DueDate == nil {
		t.Error("expected non-nil DueDate")
	}
}

func TestGenerateFromBriefingJiraDedup(t *testing.T) {
	svc := NewTodoService()
	b := &models.Briefing{
		JiraTickets: []models.JiraTicket{
			{Key: "PROJ-10", Summary: "Task", Status: "To Do", Priority: "High"},
		},
	}
	svc.GenerateFromBriefing(b)
	svc.GenerateFromBriefing(b) // second call should not add duplicate
	items := svc.List()
	if len(items) != 1 {
		t.Errorf("expected 1 item after dedup, got %d", len(items))
	}
}

func TestGenerateFromBriefingJiraPriorityBlocker(t *testing.T) {
	svc := NewTodoService()
	b := &models.Briefing{
		JiraTickets: []models.JiraTicket{
			{Key: "B-1", Summary: "Blocker", Status: "To Do", Priority: "Blocker"},
		},
	}
	svc.GenerateFromBriefing(b)
	items := svc.List()
	if items[0].Priority != models.PriorityUrgent {
		t.Errorf("Blocker should map to PriorityUrgent, got %v", items[0].Priority)
	}
}

// ── Notion todo generation ────────────────────────────────────────────────────

func TestGenerateFromBriefingNotion(t *testing.T) {
	svc := NewTodoService()
	due := time.Now().AddDate(0, 0, 2)
	b := &models.Briefing{
		NotionPages: []models.NotionPage{
			{ID: "n1", Title: "Write review", Status: "In Progress", Priority: "Urgent", DueDate: &due, URL: "https://notion.so/n1"},
			{ID: "n2", Title: "Research topic", Status: "Not Started", Priority: "High"},
			{ID: "n3", Title: "Low priority task", Status: "Not Started", Priority: "Low"},
			{ID: "n4", Title: "No priority", Status: "Not Started"},
		},
	}
	svc.GenerateFromBriefing(b)
	items := svc.List()
	if len(items) != 4 {
		t.Fatalf("expected 4 notion todos, got %d", len(items))
	}

	byID := make(map[string]models.TodoItem)
	for _, item := range items {
		byID[item.SourceID] = item
	}

	if byID["n1"].Priority != models.PriorityUrgent {
		t.Errorf("Urgent should map to PriorityUrgent, got %v", byID["n1"].Priority)
	}
	if byID["n2"].Priority != models.PriorityHigh {
		t.Errorf("High should map to PriorityHigh, got %v", byID["n2"].Priority)
	}
	if byID["n3"].Priority != models.PriorityLow {
		t.Errorf("Low should map to PriorityLow, got %v", byID["n3"].Priority)
	}
	if byID["n4"].Priority != models.PriorityMedium {
		t.Errorf("no priority should default to PriorityMedium, got %v", byID["n4"].Priority)
	}

	n1 := byID["n1"]
	if n1.Source != "notion" {
		t.Errorf("expected source=notion, got %s", n1.Source)
	}
	if n1.DueDate == nil {
		t.Error("expected non-nil DueDate")
	}
	if n1.SourceURL != "https://notion.so/n1" {
		t.Errorf("unexpected SourceURL: %s", n1.SourceURL)
	}
}

func TestGenerateFromBriefingNotionDedup(t *testing.T) {
	svc := NewTodoService()
	b := &models.Briefing{
		NotionPages: []models.NotionPage{
			{ID: "np-1", Title: "Task", Status: "Not Started", Priority: "Medium"},
		},
	}
	svc.GenerateFromBriefing(b)
	svc.GenerateFromBriefing(b)
	items := svc.List()
	if len(items) != 1 {
		t.Errorf("expected 1 item after dedup, got %d", len(items))
	}
}

func TestGenerateFromBriefingNotionCriticalPriority(t *testing.T) {
	svc := NewTodoService()
	b := &models.Briefing{
		NotionPages: []models.NotionPage{
			{ID: "c1", Title: "Critical task", Status: "Not Started", Priority: "Critical"},
		},
	}
	svc.GenerateFromBriefing(b)
	items := svc.List()
	if items[0].Priority != models.PriorityUrgent {
		t.Errorf("Critical should map to PriorityUrgent, got %v", items[0].Priority)
	}
}
